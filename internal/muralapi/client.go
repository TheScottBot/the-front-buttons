// Package muralapi is the adapter that talks to Mural's public API, as
// developers.mural.co documents it: signing in with OAuth's authorisation
// code grant, and keeping an access token, which lasts fifteen minutes, fresh
// with the refresh token signing in gives. Mural's client ID, client secret
// and refresh token live in the credential store, each bound to app.mural.co,
// and are sent nowhere else; none ever appears in an error, a log or the
// editor.
package muralapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// Mural's addresses and the scopes the keys need: reading a board's state, and
// changing it.
const (
	defaultBaseURL    = "https://" + murals.Host
	authorisationPath = "/api/public/v1/authorization/oauth2/"
	tokenPath         = "/api/public/v1/authorization/oauth2/token" //nolint:gosec // an address, not a credential
	keysScope         = "murals:read murals:write"
)

// maximumResponseBytes bounds an answer before it is parsed; a token answer
// is a few kilobytes.
const maximumResponseBytes = 1 << 16

// expiryMargin is how long before an access token runs out it is replaced,
// so one never runs out on its way to Mural.
const expiryMargin = time.Minute

// reasonPattern is the shape of an OAuth error code, such as invalid_grant;
// anything else Mural says is not repeated, as it may echo a token.
var reasonPattern = regexp.MustCompile(`^[a-z_]{1,40}$`)

// errMalformed reports an answer that is not the tokens Mural documents.
var errMalformed = errors.New("not the tokens expected")

// Client is Mural's API, for one person's sign in.
type Client struct {
	httpClient  *http.Client
	credentials platform.CredentialStore
	baseURL     string
	now         func() time.Time

	mutex       sync.Mutex
	accessToken string
	expires     time.Time
}

// NewClient returns a client for app.mural.co.
func NewClient(httpClient *http.Client, credentials platform.CredentialStore) *Client {
	return NewClientAt(httpClient, credentials, defaultBaseURL, time.Now)
}

// NewClientAt returns a client for Mural at a base address, reading the time
// from now.
func NewClientAt(
	httpClient *http.Client, credentials platform.CredentialStore, baseURL string, now func() time.Time,
) *Client {
	return &Client{
		httpClient: httpClient, credentials: credentials, baseURL: strings.TrimSuffix(baseURL, "/"), now: now,
		mutex: sync.Mutex{}, accessToken: "", expires: time.Time{},
	}
}

// AuthorizationURL is where signing in starts: Mural's own page, which asks
// the person to let the app use the timer, then returns to the redirect
// address with a code and the state given.
func (client *Client) AuthorizationURL(redirectURI string, state string) (string, error) {
	clientID, readError := client.secret(murals.ClientIDCredential, murals.ErrNotSetUp)
	if readError != nil {
		return "", readError
	}

	query := url.Values{
		"client_id": {clientID}, "redirect_uri": {redirectURI}, "scope": {keysScope},
		"state": {state}, "response_type": {"code"},
	}

	return client.baseURL + authorisationPath + "?" + query.Encode(), nil
}

// Connect trades the code Mural returned for tokens, keeping the refresh
// token in the credential store and the access token in memory.
func (client *Client) Connect(connectContext context.Context, code string, redirectURI string) error {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	tokens, tokenError := client.requestTokens(connectContext, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI},
	}, murals.ErrRefused)
	if tokenError != nil {
		return tokenError
	}

	if tokens.RefreshToken == "" {
		return fmt.Errorf("%w: signing in gave no refresh token", murals.ErrUnreachable)
	}

	return client.keep(tokens)
}

// AccessToken is a current access token, refreshed when the last has run out
// or is about to.
func (client *Client) AccessToken(tokenContext context.Context) (string, error) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	if client.accessToken != "" && client.now().Before(client.expires.Add(-expiryMargin)) {
		return client.accessToken, nil
	}

	refreshToken, readError := client.secret(murals.RefreshTokenCredential, murals.ErrNotConnected)
	if readError != nil {
		return "", readError
	}

	tokens, tokenError := client.requestTokens(tokenContext, url.Values{
		"grant_type": {"refresh_token"}, "refresh_token": {refreshToken},
	}, murals.ErrNotConnected)
	if tokenError != nil {
		return "", tokenError
	}

	keepError := client.keep(tokens)
	if keepError != nil {
		return "", keepError
	}

	return client.accessToken, nil
}

// tokenAnswer is the token endpoint's answer, as Mural documents it, in its
// own snake case.
//
//nolint:tagliatelle // the names are Mural's, not this project's
type tokenAnswer struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// errorAnswer is the token endpoint's answer to a request it refuses.
type errorAnswer struct {
	Error string `json:"error"`
}

// requestTokens posts a grant to the token endpoint with the app's client ID
// and secret in the form, as Mural documents. A grant Mural refuses fails
// with refused, by its reason.
func (client *Client) requestTokens(
	requestContext context.Context, form url.Values, refused error,
) (tokenAnswer, error) {
	clientID, idError := client.secret(murals.ClientIDCredential, murals.ErrNotSetUp)
	if idError != nil {
		return tokenAnswer{}, idError
	}

	clientSecret, secretError := client.secret(murals.ClientSecretCredential, murals.ErrNotSetUp)
	if secretError != nil {
		return tokenAnswer{}, secretError
	}

	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)

	request, requestError := http.NewRequestWithContext(requestContext, http.MethodPost,
		client.baseURL+tokenPath, strings.NewReader(form.Encode()))
	if requestError != nil {
		return tokenAnswer{}, fmt.Errorf("%w: building the request", murals.ErrUnreachable)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, sendError := client.httpClient.Do(request)
	if sendError != nil {
		return tokenAnswer{}, fmt.Errorf("%w: %w", murals.ErrUnreachable, errors.Unwrap(sendError))
	}

	defer func() { _ = response.Body.Close() }()

	body, readError := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes))
	if readError != nil {
		return tokenAnswer{}, fmt.Errorf("%w: reading the answer", murals.ErrUnreachable)
	}

	if response.StatusCode != http.StatusOK {
		return tokenAnswer{}, refusal(response.StatusCode, body, refused)
	}

	var tokens tokenAnswer

	decodeError := json.Unmarshal(body, &tokens)
	if decodeError != nil || tokens.AccessToken == "" || tokens.ExpiresIn <= 0 {
		return tokenAnswer{}, fmt.Errorf("%w: %w", murals.ErrUnreachable, errMalformed)
	}

	return tokens, nil
}

// refusal names a refused grant by its OAuth error code, or its status when
// Mural gives none; nothing else from the answer is repeated.
func refusal(status int, body []byte, refused error) error {
	var answer errorAnswer

	if json.Unmarshal(body, &answer) == nil && reasonPattern.MatchString(answer.Error) {
		return fmt.Errorf("%w: %s", refused, answer.Error)
	}

	if status >= http.StatusInternalServerError {
		return fmt.Errorf("%w: status %d", murals.ErrUnreachable, status)
	}

	return fmt.Errorf("%w: status %d", refused, status)
}

// keep holds the access token until it runs out, and stores a refresh token,
// bound to Mural, when one is given.
func (client *Client) keep(tokens tokenAnswer) error {
	client.accessToken = tokens.AccessToken
	client.expires = client.now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	if tokens.RefreshToken == "" {
		return nil
	}

	writeError := client.credentials.WriteCredential(murals.RefreshTokenCredential, tokens.RefreshToken)
	if writeError == nil {
		writeError = platform.BindCredentialHost(client.credentials, murals.RefreshTokenCredential, murals.Host)
	}

	if writeError != nil {
		return fmt.Errorf("keeping Mural's sign in: %w", writeError)
	}

	return nil
}

// secret reads one of Mural's credentials, refusing one bound to any host
// but Mural's; a credential not stored fails with missing.
func (client *Client) secret(credentialName string, missing error) (string, error) {
	secret, readError := client.credentials.ReadCredential(credentialName)

	switch {
	case errors.Is(readError, platform.ErrCredentialMissing):
		return "", fmt.Errorf("%w: %s is not stored", missing, credentialName)
	case readError != nil:
		return "", fmt.Errorf("reading %s: %w", credentialName, readError)
	}

	host, hostError := platform.CredentialHost(client.credentials, credentialName)
	if hostError != nil || host != murals.Host {
		return "", fmt.Errorf("%w: %s is not stored for %s", platform.ErrCredentialHost, credentialName, murals.Host)
	}

	return secret, nil
}
