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
// starting posts a duration and whether its sound plays; adding patches the
// time to add, in seconds, and pausing patches whether it is paused; stopping
// posts to its end, with no body; reading gets it.
const (
	muralsPath     = "/api/public/v1/murals/"
	timerPath      = "/timer"
	timerStartPath = "/timer/start"
	timerEndPath   = "/timer/end"
)

// invalidScopeCode is Mural's code for a sign in that was not given the scope
// a request needs, as one made before the keys read boards is not: it is
// mended by signing in again.
const invalidScopeCode = "INVALID_SCOPE"

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

// pauseBody is what pausing or resuming a timer patches.
type pauseBody struct {
	Paused bool `json:"paused"`
}

// timerAnswer is the timer as Mural gives it: the seconds it has left, and
// when it was paused, null if it is not. A timer never started has neither.
type timerAnswer struct {
	Value struct {
		RemainingSeconds *float64 `json:"remainingSeconds"`
		PausedTimestamp  *float64 `json:"pausedTimestamp"`
	} `json:"value"`
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

// StopTimer stops a mural's timer.
func (client *Client) StopTimer(timerContext context.Context, muralID string) error {
	return client.sendTimer(timerContext, http.MethodPost, muralID, timerEndPath, nil)
}

// PauseTimer pauses a mural's running timer, or resumes it when paused is
// false.
func (client *Client) PauseTimer(timerContext context.Context, muralID string, paused bool) error {
	return client.sendTimer(timerContext, http.MethodPatch, muralID, timerPath, pauseBody{Paused: paused})
}

// ReadTimer reads a mural's timer as it is now. It is running while it has
// time left, and paused when Mural gives the time it was paused.
func (client *Client) ReadTimer(timerContext context.Context, muralID string) (murals.TimerState, error) {
	answer, callError := client.call(timerContext, http.MethodGet, muralsPath+muralID+timerPath, muralID, nil)
	if callError != nil {
		return murals.TimerState{}, callError
	}

	var timer timerAnswer

	decodeError := json.Unmarshal(answer, &timer)
	if decodeError != nil {
		return murals.TimerState{}, fmt.Errorf("%w: the answer is not a timer", murals.ErrUnreachable)
	}

	remaining := 0.0
	if timer.Value.RemainingSeconds != nil {
		remaining = *timer.Value.RemainingSeconds
	}

	if remaining <= 0 {
		return murals.TimerState{Running: false, Paused: false, Remaining: 0}, nil
	}

	return murals.TimerState{
		Running:   true,
		Paused:    timer.Value.PausedTimestamp != nil,
		Remaining: time.Duration(remaining) * time.Second,
	}, nil
}

// sendTimer sends one timer request that changes the timer.
func (client *Client) sendTimer(
	timerContext context.Context, method string, muralID string, path string, body any,
) error {
	_, callError := client.call(timerContext, method, muralsPath+muralID+path, muralID, body)

	return callError
}

// call sends one request about a mural with a current access token, and
// returns Mural's answer when it did what was asked. A body of nil sends no
// body. An access token Mural no longer takes is dropped and the request sent
// once more with a fresh one; refused again, the sign in is no longer good.
func (client *Client) call(
	callContext context.Context, method string, path string, muralID string, body any,
) ([]byte, error) {
	if !murals.IsID(muralID) {
		return nil, fmt.Errorf("%w: not a mural ID", murals.ErrNotFound)
	}

	var encoded []byte

	if body != nil {
		marshalled, encodeError := json.Marshal(body)
		if encodeError != nil {
			return nil, fmt.Errorf("%w: encoding the request", murals.ErrUnreachable)
		}

		encoded = marshalled
	}

	for attempt := range 2 {
		token, tokenError := client.AccessToken(callContext)
		if tokenError != nil {
			return nil, tokenError
		}

		status, answer, sendError := client.send(callContext, method, path, token, encoded)
		if sendError != nil {
			return nil, sendError
		}

		if status != http.StatusUnauthorized {
			return answer, requestOutcome(status, answer)
		}

		client.forgetAccessToken()

		if attempt > 0 {
			break
		}
	}

	return nil, fmt.Errorf("%w: Mural refused the sign in", murals.ErrNotConnected)
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

	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

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

// requestOutcome reads a request's answer: done, the mural not found, a sign
// in without the scope the request needs, or refused by Mural's code.
func requestOutcome(status int, answer []byte) error {
	var refusal refusalAnswer

	code := "status " + http.StatusText(status)
	if json.Unmarshal(answer, &refusal) == nil && codePattern.MatchString(refusal.Code) {
		code = refusal.Code
	}

	switch {
	case isSuccess(status):
		return nil
	case status == http.StatusForbidden && code == invalidScopeCode:
		return fmt.Errorf("%w: %s; connect to Mural again", murals.ErrNotConnected, code)
	case status == http.StatusNotFound:
		return fmt.Errorf("%w: %s", murals.ErrNotFound, code)
	case status >= http.StatusInternalServerError:
		return fmt.Errorf("%w: %s", murals.ErrUnreachable, code)
	default:
		return fmt.Errorf("%w: %s", murals.ErrRefused, code)
	}
}

// isSuccess reports a status Mural answers a request it did with: 200, 201
// with what it made, or 204 with nothing to say.
func isSuccess(status int) bool {
	return status == http.StatusOK || status == http.StatusCreated || status == http.StatusNoContent
}

// forgetAccessToken drops the access token, so the next request refreshes it.
func (client *Client) forgetAccessToken() {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	client.accessToken = ""
}
