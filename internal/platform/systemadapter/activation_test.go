package systemadapter_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
)

// The wait and check intervals of the wait tests.
const (
	testFrontWait  = 500 * time.Millisecond
	testFrontCheck = 20 * time.Millisecond
)

// frontAfter is in front once it has been asked a number of times.
func frontAfter(asks int, later systemadapter.FrontState) func() systemadapter.FrontState {
	asked := 0

	return func() systemadapter.FrontState {
		asked++
		if asked > asks {
			return later
		}

		return systemadapter.FrontNotYet
	}
}

// TestWaitUntilInFront_WaitsForTheApplication: key presses wait until the
// application is in front, or until the system will not say, and are refused
// if it never comes forward or the press is abandoned.
func TestWaitUntilInFront_WaitsForTheApplication(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		inFront  func() systemadapter.FrontState
		wantWait time.Duration
		wantErr  error
	}{
		"in front at once":       {frontAfter(0, systemadapter.FrontYes), 0, nil},
		"in front after a while": {frontAfter(3, systemadapter.FrontYes), 3 * testFrontCheck, nil},
		"the system will not say": {
			frontAfter(1, systemadapter.FrontUnknown), testFrontCheck, nil,
		},
		"never in front": {frontAfter(1000, systemadapter.FrontYes), testFrontWait, platform.ErrApplicationNotInFront},
	}

	for name, testCase := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			synctest.Test(t, func(t *testing.T) {
				started := time.Now()

				waitError := systemadapter.WaitUntilInFront(t.Context(), "com.example.app", testCase.inFront,
					testFrontWait, testFrontCheck)

				if !errors.Is(waitError, testCase.wantErr) || (testCase.wantErr == nil && waitError != nil) {
					t.Fatalf("want %v, got %v", testCase.wantErr, waitError)
				}

				if waited := time.Since(started); waited != testCase.wantWait {
					t.Errorf("want a wait of %s, got %s", testCase.wantWait, waited)
				}
			})
		})
	}
}

// TestWaitUntilInFront_StopsWhenThePressIsAbandoned: a press abandoned while
// waiting stops waiting.
func TestWaitUntilInFront_StopsWhenThePressIsAbandoned(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		waitContext, cancel := context.WithTimeout(t.Context(), 3*testFrontCheck)
		defer cancel()

		waitError := systemadapter.WaitUntilInFront(waitContext, "com.example.app",
			frontAfter(1000, systemadapter.FrontYes), testFrontWait, testFrontCheck)
		if !errors.Is(waitError, context.DeadlineExceeded) {
			t.Fatalf("want the abandoned press reported, got %v", waitError)
		}
	})
}
