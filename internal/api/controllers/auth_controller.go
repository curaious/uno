package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/curaious/uno/internal/api/authenticator"
	"github.com/curaious/uno/internal/services"
	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"golang.org/x/oauth2"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type UserResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func RegisterAuthRoutes(r *router.Router, svc *services.Services, auth *authenticator.Authenticator) {
	r.GET("/api/agent-server/auth/enabled", func(ctx *fasthttp.RequestCtx) {
		writeResponse(ctx, requestContext(ctx), "success", map[string]any{
			"auth_enabled":  auth.AuthEnabled(),
			"auth0_enabled": auth.Auth0Enabled(),
		})
	})

	// Login with email/password
	r.POST("/api/agent-server/auth/login", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		var req LoginRequest
		if err := sonic.Unmarshal(ctx.PostBody(), &req); err != nil {
			writeError(ctx, stdCtx, "Invalid request body", err)
			return
		}

		if req.Email == "" || req.Password == "" {
			writeError(ctx, stdCtx, "Email and password are required", errors.New("missing credentials"))
			return
		}

		// Authenticate user
		user, err := svc.User.Authenticate(stdCtx, req.Email, req.Password)
		if err != nil {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			writeError(ctx, stdCtx, "Invalid credentials", err)
			return
		}

		// Generate JWT token
		token, err := auth.GenerateToken(user.ID, user.Email, user.Name, string(user.Role))
		if err != nil {
			writeError(ctx, stdCtx, "Failed to generate token", err)
			return
		}

		// Set token as HTTP-only cookie
		var cookie fasthttp.Cookie
		cookie.SetKey("access_token")
		cookie.SetValue(token)
		cookie.SetPath("/")
		cookie.SetHTTPOnly(true)
		cookie.SetSecure(false) // Set to true in production (HTTPS)
		cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
		cookie.SetExpire(time.Now().Add(24 * time.Hour))
		ctx.Response.Header.SetCookie(&cookie)

		writeResponse(ctx, stdCtx, "success", LoginResponse{
			Token: token,
			User: UserResponse{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
				Role:  string(user.Role),
			},
		})
	})

	// Get current user info
	r.GET("/api/agent-server/auth/me", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		claims, ok := ctx.UserValue("userClaims").(*authenticator.UserClaims)
		if !ok || claims == nil {
			ctx.SetStatusCode(fasthttp.StatusUnauthorized)
			writeError(ctx, stdCtx, "Unauthorized", errors.New("no user claims"))
			return
		}

		// Fetch full user from database
		user, err := svc.User.GetByID(stdCtx, claims.UserID)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to get user", err)
			return
		}

		writeResponse(ctx, stdCtx, "success", UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  string(user.Role),
		})
	})

	// Logout endpoint
	r.POST("/api/agent-server/auth/logout", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		// Clear the access_token cookie
		var cookie fasthttp.Cookie
		cookie.SetKey("access_token")
		cookie.SetValue("")
		cookie.SetPath("/")
		cookie.SetHTTPOnly(true)
		cookie.SetExpire(time.Now().Add(-1 * time.Hour))
		ctx.Response.Header.SetCookie(&cookie)

		writeResponse(ctx, stdCtx, "success", map[string]any{
			"message": "Logged out successfully",
		})
	})

	r.GET("/api/agent-server/auth/auth0/login", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)

		csrf := make([]byte, 16)
		rand.Read(csrf)

		state := authenticator.OAuthState{
			CSRF:      base64.RawURLEncoding.EncodeToString(csrf),
			Redirect:  "http://localhost:3000",
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
		}

		encodedState, err := auth.GetSignedState(state)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to create signed state", err)
			return
		}

		url := auth.AuthCodeURL(encodedState, oauth2.SetAuthURLParam("audience", auth.Audience()))
		ctx.Redirect(url, fasthttp.StatusTemporaryRedirect)
	})

	r.GET("/api/agent-server/auth/auth0/callback", func(ctx *fasthttp.RequestCtx) {
		stdCtx := requestContext(ctx)
		encodedState := ctx.URI().QueryArgs().Peek("state")
		code := ctx.URI().QueryArgs().Peek("code")

		if encodedState == nil || code == nil {
			writeError(ctx, stdCtx, "missing parameters", errors.New("missing parameters"))
			return
		}

		state, err := auth.VerifySignedState(string(encodedState))
		if err != nil {
			writeError(ctx, stdCtx, "Failed to decode state", err)
			return
		}

		token, err := auth.Exchange(stdCtx, string(code))
		if err != nil {
			writeError(ctx, stdCtx, "Failed to exchange token", err)
			return
		}

		idToken, err := auth.VerifyIDToken(stdCtx, token)
		if err != nil {
			writeError(ctx, stdCtx, "Failed to verify ID token", err)
			return
		}

		var profile map[string]interface{}
		if err := idToken.Claims(&profile); err != nil {
			writeError(ctx, stdCtx, "Failed to get claims", err)
			return
		}

		// Create cookie
		var cookie fasthttp.Cookie
		cookie.SetKey("access_token")
		cookie.SetValue(token.AccessToken)
		cookie.SetPath("/")
		cookie.SetHTTPOnly(true)
		cookie.SetSecure(false) // MUST be true in production (HTTPS)
		cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
		cookie.SetExpire(time.Now().Add(1 * time.Hour))
		ctx.Response.Header.SetCookie(&cookie)

		ctx.Redirect(state.Redirect, fasthttp.StatusFound)
	})
}
