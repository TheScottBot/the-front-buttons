package editor

import (
	"context"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The installed applications change rarely and take a moment to list (on
// macOS, one plutil run per bundle), so the list is kept for a while and
// fetched in the background, and a key's page never waits long for it.
const (
	applicationListLifetime = 10 * time.Minute
	applicationListTimeout  = 30 * time.Second
	applicationListWait     = 2 * time.Second
)

// applicationCache holds the installed applications for the app picker the
// author asked for (evaluation, deck testing). A failed listing keeps the last
// list rather than offering none.
type applicationCache struct {
	catalogue platform.ApplicationCatalogue

	mutex      sync.Mutex
	choices    []fieldChoice
	fetchedAt  time.Time
	refreshing chan struct{}
}

func newApplicationCache(catalogue platform.ApplicationCatalogue) *applicationCache {
	cache := &applicationCache{
		catalogue: catalogue, mutex: sync.Mutex{}, choices: nil, fetchedAt: time.Time{}, refreshing: nil,
	}

	return cache
}

// warm starts a listing, without waiting, if the list is missing or old. The
// grid page calls it, so the list is usually ready before a key is opened.
func (cache *applicationCache) warm(requestContext context.Context) {
	if cache.catalogue == nil {
		return
	}

	cache.mutex.Lock()
	stale := time.Since(cache.fetchedAt) >= applicationListLifetime
	cache.mutex.Unlock()

	if stale {
		cache.refresh(requestContext)
	}
}

// refresh starts one background listing unless one is already running, and
// returns a channel closed when it ends. The listing outlives the request that
// started it, so it keeps the request's values but not its cancellation.
func (cache *applicationCache) refresh(requestContext context.Context) <-chan struct{} {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if cache.refreshing != nil {
		return cache.refreshing
	}

	done := make(chan struct{})
	cache.refreshing = done

	go func() {
		defer close(done)

		listContext, cancel := context.WithTimeout(context.WithoutCancel(requestContext), applicationListTimeout)
		defer cancel()

		applications, listError := cache.catalogue.Applications(listContext)

		cache.mutex.Lock()
		defer cache.mutex.Unlock()

		cache.refreshing = nil

		if listError != nil {
			return
		}

		cache.choices = make([]fieldChoice, 0, len(applications))
		for _, application := range applications {
			cache.choices = append(cache.choices, fieldChoice{Value: application.Identifier, Label: application.Name})
		}

		cache.fetchedAt = time.Now()
	}()

	return done
}

// current returns the list, refreshing it if it is old. A page asked for
// before the first listing has finished waits a little for it, and otherwise
// shows the field without a picker.
func (cache *applicationCache) current(requestContext context.Context) []fieldChoice {
	if cache.catalogue == nil {
		return nil
	}

	cache.mutex.Lock()
	choices, fetchedAt := cache.choices, cache.fetchedAt
	cache.mutex.Unlock()

	if time.Since(fetchedAt) < applicationListLifetime {
		return choices
	}

	done := cache.refresh(requestContext)

	if choices == nil {
		select {
		case <-done:
		case <-time.After(applicationListWait):
		case <-requestContext.Done():
		}

		cache.mutex.Lock()
		choices = cache.choices
		cache.mutex.Unlock()
	}

	return choices
}

// attachChoices fills each field's picker from the list its schema names.
func (pages *pageHandlers) attachChoices(requestContext context.Context, fields []fieldView) {
	for index := range fields {
		if fields[index].Suggestions == brick.SuggestApplications && fields[index].Control == textControl {
			fields[index].Choices = pages.apps.current(requestContext)
		}
	}
}
