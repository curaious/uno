package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/fasthttp/router"
	"github.com/praveen001/uno/internal/api/authenticator"
	"github.com/praveen001/uno/internal/services"
	"github.com/valyala/fasthttp"
	"golang.org/x/oauth2"
)

func RegisterAuthRoutes(r *router.Router, svc *services.Services, auth *authenticator.Authenticator) {
	r.GET("/api/auth/enabled", func(ctx *fasthttp.RequestCtx) {
		writeResponse(ctx, requestContext(ctx), "success", map[string]any{
			"auth_enabled": auth.AuthEnabled(),
		})
	})

	r.GET("/api/auth/login", func(ctx *fasthttp.RequestCtx) {
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

	r.GET("/api/auth/callback", func(ctx *fasthttp.RequestCtx) {
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
