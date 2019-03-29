package main

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/concourse/concourse/go-concourse/concourse"
	"golang.org/x/oauth2"
)

func NewHttpClient(concourseURL string, username string, password string) (*http.Client, error) {
	tokenEndPoint, err := url.Parse("sky/token")
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(concourseURL)
	if err != nil {
		return nil, err
	}

	tokenURL := base.ResolveReference(tokenEndPoint)

	oauth2Config := oauth2.Config{
		ClientID:     "fly",
		ClientSecret: "Zmx5",
		Endpoint:     oauth2.Endpoint{TokenURL: tokenURL.String()},
		Scopes:       []string{"openid", "profile", "email", "federated:id", "groups"},
	}

	httpClient := &http.Client{Timeout: 2 * time.Second}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, &httpClient)

	token, err := oauth2Config.PasswordCredentialsToken(ctx, username, password)
	if err != nil {
		return nil, err
	}

	return oauth2Config.Client(ctx, token), nil
}

func NewConcourseClient(concourseURL string, username string, password string) (concourse.Client, error) {
	httpClient, err := NewHttpClient(concourseURL, username, password)
	if err != nil {
		return nil, err
	}

	client := concourse.NewClient(concourseURL, httpClient, false)

	return client, nil
}
