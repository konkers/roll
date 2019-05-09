package roll

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"time"

	oidc "github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
	twitchoauth "golang.org/x/oauth2/twitch"
)

// Auth strategy:
//
// Authentication is done through twitch OIDC.
// Auth token is stored in gorilla session key "auth":
//  session.Values['subject'] = twitter oidc subject aka user ID
//  session.Values['expiry'] = time.Time of expiration
//
// Todo:
//   Support state and none.
//   Support redirecting back on auth failure

func (b *Bot) authUpdateToken(subject string, w http.ResponseWriter, r *http.Request) error {
	session, _ := b.sessionStore.Get(r, "auth")

	session.Values["subject"] = subject
	session.Values["expiry"] = time.Now().Add(time.Hour * 12)
	log.Printf("updating session: %#v", session.Values)
	return session.Save(r, w)
}

func (b *Bot) authCheckToken(w http.ResponseWriter, r *http.Request) (string, error) {
	session, _ := b.sessionStore.Get(r, "auth")

	// First check for valid subject and
	var subject string
	var ok bool
	log.Printf("session: %#v", session.Values)
	subject, ok = session.Values["subject"].(string)
	if !ok {
		return "", fmt.Errorf("No subject key in session")
	}

	var expiry time.Time
	expiry, ok = session.Values["expiry"].(time.Time)
	if !ok {
		return "", fmt.Errorf("No expiry key in session")
	}

	if expiry.Before(time.Now()) {
		return "", fmt.Errorf("Session Expired")
	}

	return subject, b.authUpdateToken(subject, w, r)
}

func (b *Bot) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		subject, err := b.authCheckToken(w, r)

		if err == nil {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "subject", subject)))
		} else {
			log.Printf("Auth error: %v", err)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "subject", "")))
		}
	})
}

func (b *Bot) authInit() {
	// Register time.Time with gob so for session expiry
	gob.Register(time.Time{})

	var err error
	b.authProvider, err = oidc.NewProvider(context.TODO(), "https://id.twitch.tv/oauth2")
	if err != nil {
		log.Fatalf("Can't create auth provider %s: %v", twitchoauth.Endpoint.AuthURL, err)
	}

	b.authVerifier = b.authProvider.Verifier(&oidc.Config{ClientID: b.Config.ClientID})

	b.authConfig = &oauth2.Config{
		ClientID:     b.Config.ClientID,
		ClientSecret: b.Config.ClientSecret,
		RedirectURL:  fmt.Sprintf("https://%s/auth", b.Config.HTTPSAddr),

		// Discovery returns the OAuth2 endpoints.
		Endpoint: b.authProvider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "user:read:email"},
	}
	// Configure an OpenID Connect aware OAuth2 client.
}

func (b *Bot) authUserHandler(w http.ResponseWriter, req *http.Request) {
	log.Printf("redir: %s", b.authConfig.AuthCodeURL("teststate"))
	http.Redirect(w, req, b.authConfig.AuthCodeURL("teststate"), http.StatusFound)
}

func (b *Bot) authHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	authCode := r.URL.Query().Get("code")
	log.Printf("code: %v", authCode)
	if authCode != "" {
		oauth2Token, err := b.authConfig.Exchange(ctx, authCode)
		if err != nil {
			log.Printf("exchange error %v: %v", authCode, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
			// handle error
		}

		// Extract the ID Token from OAuth2 token.
		rawIDToken, ok := oauth2Token.Extra("id_token").(string)
		if !ok {
			log.Printf("extra error")
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
			// handle missing token
		}

		// Parse and verify ID Token payload.
		idToken, err := b.authVerifier.Verify(ctx, rawIDToken)
		if err != nil {
			log.Printf("verify error: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			return
			// handle error
		}

		log.Printf("token: %#v", idToken.Subject)
		err = b.authUpdateToken(idToken.Subject, w, r)
		if err != nil {
			log.Printf("error updating token: %v", err)
		}
	}
	b.execTemplate("auth.html", w, r)
}
