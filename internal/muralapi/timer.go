package muralapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// The timer's addresses, under a mural's own, as Mural documents them:
// starting posts a duration and whether its sound plays, and adding patches
// the time to add, both in seconds.
const (
	muralsPath     = "/api/public/v1/murals/"
	timerPath      = "/timer"
	timerStartPath = "/timer/start"
)

// codePattern is the shape of the code Mural names a refusal by, such as
// MURAL_NOT_FOUND; nothing else it says is repeated.
var codePattern = regexp.MustCompile(`^[A-Z_]{1,40}$`)

// startBody is what starting a timer posts.
type startBody struct {
	Duration     int64 `json:"duration"`
	SoundEnabled bool  `json:"soundEnabled"`
}

// addBody is what adding to a timer patches.
type addBody struct {
	Delta int64 `json:"delta"`
}

// refusalAnswer is how Mural names a request it refuses.
type refusalAnswer struct {
	Code string `json:"code"`
}

// StartTimer starts a mural's timer for a duration, with its sound at the end
// or not.
func (client *Client) StartTimer(
	timerContext context.Context, muralID string, duration time.Duration, sound bool,
) error {
	return client.sendTimer(timerContext, http.MethodPost, muralID, timerStartPath,
		startBody{Duration: int64(duration / time.Second), SoundEnabled: sound})
}

// AddTime adds time to a mural's running timer.
func (client *Client) AddTime(timerContext context.Context, muralID string, extra time.Duration) error {
	return client.sendTimer(timerContext, http.MethodPatch, muralID, timerPath,
		addBody{Delta: int64(extra / time.Second)})
}

// sendTimer sends one timer request with a current access token. An access
// token Mural no longer takes is dropped and the request sent once more with
// a fresh one; refused again, the sign in is no longer good.
func (client *Client) sendTimer(
	timerContext context.Context, method string, muralID string, path string, body any,
) error {
	if !murals.IsID(muralID) {
		return fmt.Errorf("%w: not a mural ID", murals.ErrNotFound)
	}

	encoded, encodeError := json.Marshal(body)
	if encodeError != nil {
		return fmt.Errorf("%w: encoding the request", murals.ErrUnreachable)
	}

	for attempt := range 2 {
		token, tokenError := client.AccessToken(timerContext)
		if tokenError != nil {
			return tokenError
		}

		status, answer, sendError := client.send(timerContext, method, muralsPath+muralID+path, token, encoded)
		if sendError != nil {
			return sendError
		}

		if status != http.StatusUnauthorized {
			return timerOutcome(status, answer)
		}

		client.forgetAccessToken()

		if attempt > 0 {
			break
		}
	}

	return fmt.Errorf("%w: Mural refused the sign in", murals.ErrNotConnected)
}

// send makes one request with the access token, returning its status and a
// bounded answer.
func (client *Client) send(
	sendContext context.Context, method string, path string, token string, body []byte,
) (int, []byte, error) {
	request, requestError := http.NewRequestWithContext(sendContext, method, client.baseURL+path, bytes.NewReader(body))
	if requestError != nil {
		return 0, nil, fmt.Errorf("%w: building the request", murals.ErrUnreachable)
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, sendError := client.httpClient.Do(request)
	if sendError != nil {
		return 0, nil, fmt.Errorf("%w: sending the request", murals.ErrUnreachable)
	}

	defer func() { _ = response.Body.Close() }()

	answer, readError := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes))
	if readError != nil {
		return 0, nil, fmt.Errorf("%w: reading the answer", murals.ErrUnreachable)
	}

	return response.StatusCode, answer, nil
}

// timerOutcome reads a timer request's answer: done, the mural not found, or
// refused by Mural's code.
func timerOutcome(status int, answer []byte) error {
	var refusal refusalAnswer

	code := "status " + http.StatusText(status)
	if json.Unmarshal(answer, &refusal) == nil && codePattern.MatchString(refusal.Code) {
		code = refusal.Code
	}

	switch {
	case status == http.StatusOK || status == http.StatusCreated:
		return nil
	case status == http.StatusNotFound:
		return fmt.Errorf("%w: %s", murals.ErrNotFound, code)
	case status >= http.StatusInternalServerError:
		return fmt.Errorf("%w: %s", murals.ErrUnreachable, code)
	default:
		return fmt.Errorf("%w: %s", murals.ErrRefused, code)
	}
}

// forgetAccessToken drops the access token, so the next request refreshes it.
func (client *Client) forgetAccessToken() {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.accessToken = ""
}
