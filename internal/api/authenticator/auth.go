package authenticator

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/bytedance/sonic"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/praveen001/uno/internal/config"
	"golang.org/x/oauth2"
)

type Authenticator struct {
	*oidc.Provider
	oauth2.Config

	stateSecret  string
	issuer       string
	jwksProvider *jwks.CachingProvider
	audience     string
	authEnabled  bool
}

func New(conf *config.Config) (*Authenticator, error) {
	if conf.AUTH0_DOMAIN == "" {
		return &Authenticator{
			authEnabled: false,
		}, nil
	}

	issuer := "https://" + conf.AUTH0_DOMAIN + "/"

	provider, err := oidc.NewProvider(context.Background(), issuer)
	if err != nil {
		return nil, err
	}

	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return nil, err
	}

	return &Authenticator{
		Provider: provider,
		Config: oauth2.Config{
			ClientID:     conf.AUTH0_CLIENT_ID,
			ClientSecret: conf.AUTH0_CLIENT_SECRET,
			RedirectURL:  conf.AUTH0_CALLBACK_URL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "profile"},
		},
		stateSecret:  conf.STATE_SECRET,
		issuer:       issuer,
		jwksProvider: jwks.NewCachingProvider(issuerURL, 5*time.Minute),
		audience:     "uno-api",
	}, nil
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

func (a *Authenticator) VerifyAccessToken(ctx context.Context, token string) error {
	jwtValidator, err := validator.New(a.jwksProvider.KeyFunc, validator.RS256, a.issuer, []string{a.Audience()})
	if err != nil {
		return err
	}

	payload, err := jwtValidator.ValidateToken(ctx, token)
	if err != nil {
		return err
	}

	fmt.Println(payload)

	return nil
}
