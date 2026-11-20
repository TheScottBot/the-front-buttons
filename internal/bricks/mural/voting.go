package mural

import (
	"context"
	"fmt"
	"image/color"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/murals"
	"github.com/TheScottBot/the-front-buttons/internal/render"
)

// The voting key, mural_voting, runs a voting session on a board: a press
// starts one, the next ends it. While it runs the key shows the votes cast;
// once it ends, the most votes any one widget has. Mural's API cannot say
// which session runs, so the keys remember the session they started, from
// one press to the next while the daemon runs.

// VotingKindName is the voting key's name in configuration.
const VotingKindName = "mural_voting"

// votingIdentity names the live source a board's voting keys share.
const votingIdentity = "mural_voting:"

// votingRead is how often the key reads a running session's results.
const votingRead = 10 * time.Second

// The voting key's defaults: the session's title, and the votes each person
// has.
const (
	defaultVotingTitle = "Vote"
	defaultVotes       = 3
)

// The voting key's icons: a check box while the session runs, a trophy once
// it ends.
const (
	votingOpenIcon  = "checkbox"
	votingEndedIcon = "trophy"
)

// votingOpenBackground lights the key while its session runs.
//
//nolint:gochecknoglobals // a colour, never changed
var votingOpenBackground = color.RGBA{R: 0x2E, G: 0x7D, B: 0x32, A: 0xFF} //nolint:mnd // a colour's own channels

// VotingParameters is the voting key's one declaration of its parameters.
type VotingParameters struct {
	brick.Appearance

	Link         string `brick:"link,required"  bounds:"1,2048"                                                   format:"mural_board"                                                           description:"The board's address, as its browser shows it; the key runs voting on that board."`
	Title        string `brick:"title"          bounds:"0,200"                                                    description:"The voting session's title, as Mural shows it; Vote if left out."`
	Votes        int    `brick:"votes"          bounds:"0,100"                                                    description:"The votes each person has; three if left out."`
	AnyoneCanEnd bool   `brick:"anyone_can_end" description:"Let anyone end the session, not only a facilitator."`
}

// ballot is what the keys know of a board's voting session: its ID, and
// whether it still runs.
type ballot struct {
	id   string
	open bool
}

// ballots holds each board's session, shared by the voting keys made from
// one registry.
type ballots struct {
	mutex    sync.Mutex
	sessions map[string]ballot
}

func newBallots() *ballots {
	return &ballots{mutex: sync.Mutex{}, sessions: map[string]ballot{}}
}

func (known *ballots) get(muralID string) (ballot, bool) {
	known.mutex.Lock()
	defer known.mutex.Unlock()

	session, found := known.sessions[muralID]

	return session, found
}

func (known *ballots) set(muralID string, session ballot) {
	known.mutex.Lock()
	defer known.mutex.Unlock()

	known.sessions[muralID] = session
}

// registerVoting adds the voting kind to a registry.
func registerVoting(registry *brick.Registry, shared *wakes, known *ballots) error {
	return brick.Register(registry, VotingKindName, func( //nolint:wrapcheck // Register already names the kind
		parameters VotingParameters, ports brick.Ports,
	) (any, error) {
		if ports.MuralBoard == nil {
			return nil, fmt.Errorf("%w: %s needs Mural's API", brick.ErrPortMissing, VotingKindName)
		}

		options := murals.VotingOptions{
			Title: parameters.Title, Votes: parameters.Votes, AnyoneCanEnd: parameters.AnyoneCanEnd,
		}

		if options.Title == "" {
			options.Title = defaultVotingTitle
		}

		if options.Votes == 0 {
			options.Votes = defaultVotes
		}

		// The link was checked as a board's address when the configuration
		// was read, so it always names one.
		muralID, _ := murals.BoardID(parameters.Link)

		return voting{
			muralID: muralID, board: ports.MuralBoard, options: options, known: known,
			wake: shared.forSource(votingIdentity + muralID),
		}, nil
	})
}

// voting is one voting key.
type voting struct {
	muralID string
	board   murals.Board
	options murals.VotingOptions
	known   *ballots
	wake    chan struct{}
}

// LiveIdentity is shared by every voting key for the board.
func (key voting) LiveIdentity() string { return votingIdentity + key.muralID }

// DefaultLabel says what a first press does.
func (voting) DefaultLabel() string { return "Start vote" }

// Watch publishes the board's session as the keys know it, reading a running
// session's results every votingRead and an ended one's once, until the
// context ends. A press wakes it to read at once.
func (key voting) Watch(watchContext context.Context, publish func(state brick.LiveState)) {
	var last brick.LiveState

	for first := true; ; first = false {
		state, wait := key.read(watchContext)
		if watchContext.Err() != nil {
			return
		}

		if first || state != last {
			publish(state)

			last = state
		}

		var timeout <-chan time.Time
		if wait > 0 {
			timeout = time.After(wait)
		}

		select {
		case <-watchContext.Done():
			return
		case <-key.wake:
		case <-timeout:
		}
	}
}

// DrawLive shows the votes on a lit key while the session runs, the top count
// with a trophy once it ends, and the key's own look before any session.
func (voting) DrawLive(state brick.LiveState, appearance *render.Appearance) {
	if state.Text == "" {
		return
	}

	appearance.Label = state.Text

	appearance.Icon = votingEndedIcon
	if state.Playing {
		appearance.Icon = votingOpenIcon
		appearance.Background = votingOpenBackground
	}
}

// PressWithState ends the board's session while it runs, and otherwise starts
// one, then has the watch read it at once.
func (key voting) PressWithState(pressContext context.Context, _ brick.LiveState) error {
	session, found := key.known.get(key.muralID)

	if found && session.open {
		endError := key.board.EndVoting(pressContext, key.muralID)
		if endError != nil {
			return votingFailed(endError)
		}

		key.known.set(key.muralID, ballot{id: session.id, open: false})
	} else {
		sessionID, startError := key.board.StartVoting(pressContext, key.muralID, key.options)
		if startError != nil {
			return votingFailed(startError)
		}

		key.known.set(key.muralID, ballot{id: sessionID, open: true})
	}

	wakeUp(key.wake)

	return nil
}

// read is the board's session as it is now, and how long until it is worth
// reading again: none while there is no session or it has ended, since only
// a press changes those.
func (key voting) read(readContext context.Context) (brick.LiveState, time.Duration) {
	state := brick.LiveState{
		Status: render.StatusNone, ErrorReason: "", TargetURL: "", Picture: nil, Playing: false, Text: "",
		Urgent: false,
	}

	session, found := key.known.get(key.muralID)
	if !found {
		return state, 0
	}

	tally, tallyError := key.board.TallyVoting(readContext, key.muralID, session.id)
	if tallyError != nil {
		state.ErrorReason = timerFailureCategory(tallyError)

		return state, countdownRetry
	}

	if !session.open {
		state.Text = fmt.Sprintf("Top %d", tally.Top)

		return state, 0
	}

	state.Playing = true

	state.Text = fmt.Sprintf("%d votes", tally.Total)
	if tally.Total == 1 {
		state.Text = "1 vote"
	}

	return state, votingRead
}

// votingFailed says why a press could not run the board's voting.
func votingFailed(failure error) error {
	return fmt.Errorf("using the board's voting: %w", brick.Categorised(timerFailureCategory(failure), failure))
}
