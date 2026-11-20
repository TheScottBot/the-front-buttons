package muralapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"

	"github.com/TheScottBot/the-front-buttons/internal/murals"
)

// Voting's addresses, under a mural's own, as Mural documents them: starting
// posts the session's title, votes each and whether anyone can end it, and
// answers with its ID; ending posts to the end with no body, ending the
// session running; and a session's results are read a page at a time.
const (
	votingStartPath   = "/voting-sessions/start"
	votingEndPath     = "/voting-sessions/end"
	votingSessionPath = "/voting-sessions/"
	votingResultsPath = "/results"
)

// The results' paging: as many results on a page as Mural takes (it wants
// fewer than 100), and a limit on the pages followed, so a next token that
// never ends cannot keep a key reading.
const (
	resultsPageSize = 99
	resultsPagesMax = 20
)

// sessionIDPattern is the shape of a voting session's ID, as Mural gives it,
// checked before it goes into an address.
var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// votingStartBody is what starting a voting session posts.
type votingStartBody struct {
	Title        string `json:"title"`
	Votes        int    `json:"numberOfVotes"`
	AnyoneCanEnd bool   `json:"anyoneCanEnd"`
}

// votingStartAnswer is the session Mural started.
type votingStartAnswer struct {
	Value struct {
		ID string `json:"id"`
	} `json:"value"`
}

// votingResultsAnswer is one page of a session's results, and the token for
// the next page, empty on the last.
type votingResultsAnswer struct {
	Value []struct {
		TotalVotes int `json:"totalVotes"`
	} `json:"value"`
	Next string `json:"next"`
}

// StartVoting starts a voting session on a mural and gives back its ID.
func (client *Client) StartVoting(
	boardContext context.Context, muralID string, options murals.VotingOptions,
) (string, error) {
	answer, callError := client.call(boardContext, http.MethodPost, muralsPath+muralID+votingStartPath, muralID,
		votingStartBody{Title: options.Title, Votes: options.Votes, AnyoneCanEnd: options.AnyoneCanEnd})
	if callError != nil {
		return "", callError
	}

	var started votingStartAnswer

	decodeError := json.Unmarshal(answer, &started)
	if decodeError != nil || !sessionIDPattern.MatchString(started.Value.ID) {
		return "", fmt.Errorf("%w: the answer is not a voting session", murals.ErrUnreachable)
	}

	return started.Value.ID, nil
}

// EndVoting ends the voting session running on a mural.
func (client *Client) EndVoting(boardContext context.Context, muralID string) error {
	_, callError := client.call(boardContext, http.MethodPost, muralsPath+muralID+votingEndPath, muralID, nil)

	return callError
}

// TallyVoting reads a voting session's results, page by page, and tallies the
// votes cast in all and the most any one widget has.
func (client *Client) TallyVoting(
	boardContext context.Context, muralID string, sessionID string,
) (murals.VotingTally, error) {
	if !sessionIDPattern.MatchString(sessionID) {
		return murals.VotingTally{}, fmt.Errorf("%w: not a voting session ID", murals.ErrNotFound)
	}

	tally := murals.VotingTally{Total: 0, Top: 0}
	next := ""

	for range resultsPagesMax {
		query := url.Values{"limit": {strconv.Itoa(resultsPageSize)}}
		if next != "" {
			query.Set("next", next)
		}

		path := muralsPath + muralID + votingSessionPath + sessionID + votingResultsPath + "?" + query.Encode()

		answer, callError := client.call(boardContext, http.MethodGet, path, muralID, nil)
		if callError != nil {
			return murals.VotingTally{}, callError
		}

		var page votingResultsAnswer

		decodeError := json.Unmarshal(answer, &page)
		if decodeError != nil {
			return murals.VotingTally{}, fmt.Errorf("%w: the answer is not voting results", murals.ErrUnreachable)
		}

		for _, result := range page.Value {
			tally.Total += result.TotalVotes
			tally.Top = max(tally.Top, result.TotalVotes)
		}

		if page.Next == "" {
			break
		}

		next = page.Next
	}

	return tally, nil
}
