package authenticator

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/bytedance/sonic"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"
	"github.com/praveen001/uno/internal/config"
	"golang.org/x/oauth2"
)

// UserClaims represents the claims stored in our JWT tokens
type UserClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

type Authenticator struct {
	*oidc.Provider
	oauth2.Config

	stateSecret  string
	jwtSecret    string
	issuer       string
	jwksProvider *jwks.CachingProvider
	audience     string
	authEnabled  bool
	auth0Enabled bool
}

func New(conf *config.Config) (*Authenticator, error) {
	auth := &Authenticator{
		stateSecret:  conf.STATE_SECRET,
		jwtSecret:    conf.JWT_SECRET,
		authEnabled:  true,
		auth0Enabled: true,
		audience:     "uno-api",
	}

	// If Auth0 is configured, set it up
	if conf.AUTH0_DOMAIN != "" {
		issuer := "https://" + conf.AUTH0_DOMAIN + "/"

		provider, err := oidc.NewProvider(context.Background(), issuer)
		if err != nil {
			return nil, err
		}

		issuerURL, err := url.Parse(issuer)
		if err != nil {
			return nil, err
		}

		auth.Provider = provider
		auth.Config = oauth2.Config{
			ClientID:     conf.AUTH0_CLIENT_ID,
			ClientSecret: conf.AUTH0_CLIENT_SECRET,
			RedirectURL:  conf.AUTH0_CALLBACK_URL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		}
		auth.issuer = issuer
		auth.jwksProvider = jwks.NewCachingProvider(issuerURL, 5*time.Minute)
		auth.auth0Enabled = true
	}

	return auth, nil
}

func (a *Authenticator) AuthEnabled() bool {
	return a.authEnabled
}

func (a *Authenticator) Audience() string {
	return a.audience
}

// VerifyIDToken verifies that an *oauth2.Token is a valid *oidc.IDToken.
func (a *Authenticator) VerifyIDToken(ctx context.Context, token *oauth2.Token) (*oidc.IDToken, error) {
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("no id_token field in oauth2 token")
	}

	oidcConfig := &oidc.Config{
		ClientID: a.ClientID,
	}

	return a.Verifier(oidcConfig).Verify(ctx, rawIDToken)
}

type OAuthState struct {
	CSRF      string `json:"csrf"`
	Redirect  string `json:"redirect"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

func (a *Authenticator) GetSignedState(state OAuthState) (string, error) {
	payload, err := sonic.Marshal(state)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, []byte(a.stateSecret))
	mac.Write(payload)
	sig := mac.Sum(nil)

	combined := append(payload, sig...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

func (a *Authenticator) VerifySignedState(encodedState string) (*OAuthState, error) {
	raw, err := base64.StdEncoding.DecodeString(encodedState)
	if err != nil {
		return nil, errors.New("invalid base64")
	}

	if len(raw) < sha256.Size {
		return nil, errors.New("state too short")
	}

	payload := raw[:len(raw)-sha256.Size]
	sig := raw[len(raw)-sha256.Size:]

	mac := hmac.New(sha256.New, []byte(a.stateSecret))
	mac.Write(payload)
	expectedSig := mac.Sum(nil)
	if !hmac.Equal(sig, expectedSig) {
		return nil, errors.New("invalid state signature")
	}

	var state OAuthState
	if err := sonic.Unmarshal(payload, &state); err != nil {
		return nil, errors.New("invalid state payload")
	}

	if time.Now().Unix() > state.ExpiresAt {
		return nil, errors.New("state expired")
	}

	return &state, nil
}

func (a *Authenticator) Auth0Enabled() bool {
	return a.auth0Enabled
}

// GenerateToken creates a new JWT token for a user
func (a *Authenticator) GenerateToken(userID, email, name, role string) (string, error) {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "uno",
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Email:  email,
		Name:   name,
		Role:   role,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.jwtSecret))
}

// VerifyLocalToken verifies a locally-issued JWT token and returns the claims
func (a *Authenticator) VerifyLocalToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.jwtSecret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// VerifyAccessToken verifies an access token (tries local JWT first, then Auth0 if enabled)
func (a *Authenticator) VerifyAccessToken(ctx context.Context, token string) (*UserClaims, error) {
	// First, try to verify as a local JWT
	claims, err := a.VerifyLocalToken(token)
	if err == nil {
		return claims, nil
	}

	// If local verification failed and Auth0 is enabled, try Auth0
	if a.auth0Enabled {
		jwtValidator, err := validator.New(a.jwksProvider.KeyFunc, validator.RS256, a.issuer, []string{a.Audience()})
		if err != nil {
			return nil, err
		}

		payload, err := jwtValidator.ValidateToken(ctx, token)
		if err != nil {
			return nil, err
		}

		// Extract claims from Auth0 token
		validatedClaims := payload.(*validator.ValidatedClaims)
		return &UserClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject: validatedClaims.RegisteredClaims.Subject,
			},
			UserID: validatedClaims.RegisteredClaims.Subject,
		}, nil
	}

	return nil, errors.New("invalid token")
}
