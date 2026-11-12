package hidtransport_test

import (
	"errors"
	"testing"

	"github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport"
)

var errTransient = errors.New("IOHIDDeviceSetReport failed: general error")

func TestRetryTransient_SucceedsWhenALaterAttemptDoes(t *testing.T) {
	t.Parallel()

	calls := 0

	retryError := hidtransport.RetryTransient(func() error {
		calls++
		if calls < hidtransport.WriteAttempts {
			return errTransient
		}

		return nil
	})
	if retryError != nil || calls != hidtransport.WriteAttempts {
		t.Fatalf("want success on attempt %d, got %v after %d calls", hidtransport.WriteAttempts, retryError, calls)
	}
}

func TestRetryTransient_GivesUpAfterEveryAttemptFails(t *testing.T) {
	t.Parallel()

	calls := 0

	retryError := hidtransport.RetryTransient(func() error {
		calls++

		return errTransient
	})
	if !errors.Is(retryError, errTransient) || calls != hidtransport.WriteAttempts {
		t.Fatalf("want the last error after %d calls, got %v after %d", hidtransport.WriteAttempts, retryError, calls)
	}
}

func TestRetryTransient_StopsAtTheFirstSuccess(t *testing.T) {
	t.Parallel()

	calls := 0

	retryError := hidtransport.RetryTransient(func() error {
		calls++

		return nil
	})
	if retryError != nil || calls != 1 {
		t.Fatalf("want one call, got %v after %d", retryError, calls)
	}
}
