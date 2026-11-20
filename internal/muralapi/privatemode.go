package muralapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// Private mode's addresses, under a mural's own, as Mural documents them:
// starting posts whether authors are hidden, ending posts to its end with no
// body, and reading gets whether it is on.
const (
	privateModePath      = "/private-mode"
	privateModeStartPath = "/private-mode/start"
	privateModeEndPath   = "/private-mode/end"
)

// privateModeStartBody is what starting private mode posts.
type privateModeStartBody struct {
	HideAuthors bool `json:"hideAuthors"`
}

// privateModeAnswer is private mode as Mural gives it: on or off.
type privateModeAnswer struct {
	Value struct {
		Active bool `json:"active"`
	} `json:"value"`
}

// StartPrivateMode starts private mode on a mural, hiding who wrote what or
// not.
func (client *Client) StartPrivateMode(boardContext context.Context, muralID string, hideAuthors bool) error {
	_, callError := client.call(boardContext, http.MethodPost, muralsPath+muralID+privateModeStartPath, muralID,
		privateModeStartBody{HideAuthors: hideAuthors})

	return callError
}

// EndPrivateMode ends private mode on a mural.
func (client *Client) EndPrivateMode(boardContext context.Context, muralID string) error {
	_, callError := client.call(boardContext, http.MethodPost, muralsPath+muralID+privateModeEndPath, muralID, nil)

	return callError
}

// ReadPrivateMode reads whether private mode is on for a mural.
func (client *Client) ReadPrivateMode(boardContext context.Context, muralID string) (bool, error) {
	answer, callError := client.call(boardContext, http.MethodGet, muralsPath+muralID+privateModePath, muralID, nil)
	if callError != nil {
		return false, callError
	}

	var mode privateModeAnswer

	decodeError := json.Unmarshal(answer, &mode)
	if decodeError != nil {
		return false, fmt.Errorf("%w: the answer is not private mode", murals.ErrUnreachable)
	}

	return mode.Value.Active, nil
}
