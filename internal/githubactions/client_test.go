package githubactions_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/githubactions"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
	"github.com/TheScottBot/the-front-buttons/internal/testsupport"
)

// The GitHub status and conclusion values the tests write, named once.
const (
	completed  = "completed"
	inProgress = "in_progress"
	succeeded  = "success"
	failed     = "failure"
)

const (
	fakeToken          = "github_pat_FAKE0123456789" //nolint:gosec // a made up token for the fake server
	fakeCredentialName = "github-work"
	runsPath           = "/repos/octo-org/hello-world/actions/workflows/ci.yml/runs"
)

// fakeGitHub is a local TLS server answering the workflow runs endpoint as the
// GitHub REST documentation describes it (evaluation section 5).
type fakeGitHub struct {
	mutex     sync.Mutex
	server    *httptest.Server
	requests  []*http.Request
	respond   func(writer http.ResponseWriter, request *http.Request)
	responses int
}

func newFakeGitHub(t *testing.T, respond func(writer http.ResponseWriter, request *http.Request)) *fakeGitHub {
	t.Helper()

	fake := &fakeGitHub{mutex: sync.Mutex{}, server: nil, requests: nil, respond: respond, responses: 0}
	fake.server = httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		fake.mutex.Lock()
		fake.requests = append(fake.requests, request.Clone(request.Context()))
		fake.responses++
		fake.mutex.Unlock()

		fake.respond(writer, request)
	}))
	t.Cleanup(fake.server.Close)

	return fake
}

func (fake *fakeGitHub) requestCount() int {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return len(fake.requests)
}

func (fake *fakeGitHub) lastRequest() *http.Request {
	fake.mutex.Lock()
	defer fake.mutex.Unlock()

	return fake.requests[len(fake.requests)-1]
}

// host is the fake's address as a URL host, with its port.
func (fake *fakeGitHub) host() string {
	return strings.TrimPrefix(fake.server.URL, "https://")
}

func (fake *fakeGitHub) query() pipeline.Query {
	return pipeline.Query{
		APIBaseURL:     fake.server.URL,
		Repository:     "octo-org/hello-world",
		Workflow:       "ci.yml",
		Branch:         "main",
		CredentialName: fakeCredentialName,
	}
}

func newClient(t *testing.T, fake *fakeGitHub) (*githubactions.Client, *diagnostics.Counters) {
	t.Helper()

	credentials := testsupport.NewCredentialStoreTestDouble()

	writeError := credentials.WriteCredential(fakeCredentialName, fakeToken)
	if writeError != nil {
		t.Fatal(writeError)
	}

	// The fake GitHub stands in for an Enterprise host, so the token is bound
	// to it as a person binds a token on the credentials page.
	bindError := platform.BindCredentialHost(credentials, fakeCredentialName, fake.host())
	if bindError != nil {
		t.Fatal(bindError)
	}

	counters := diagnostics.NewCounters()

	return githubactions.NewClient(fake.server.Client(), credentials, counters), counters
}

// runsJSON writes a workflow runs response, newest first, each run given as
// status, conclusion and number.
func runsJSON(runs ...[3]string) string {
	entries := make([]string, 0, len(runs))
	for _, run := range runs {
		conclusion := "null"
		if run[1] != "" {
			conclusion = `"` + run[1] + `"`
		}

		entries = append(entries, fmt.Sprintf(
			`{"id":%s,"status":"%s","conclusion":%s,"html_url":"%s","head_branch":"main"}`,
			run[2], run[0], conclusion, runURL(run[2])))
	}

	return fmt.Sprintf(`{"total_count":%d,"workflow_runs":[%s]}`, len(runs), strings.Join(entries, ","))
}

func answerWith(body string, headers map[string]string) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, _ *http.Request) {
		for name, value := range headers {
			writer.Header().Set(name, value)
		}

		_, _ = writer.Write([]byte(body))
	}
}

func answerStatus(status int) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, _ *http.Request) { writer.WriteHeader(status) }
}

func runURL(runNumber string) string {
	return "https://github.com/octo-org/hello-world/actions/runs/" + runNumber
}

func TestLatestRun_ReportsSuccessFailureAndInProgress(t *testing.T) {
	t.Parallel()

	stateCases := map[string]struct {
		body      string
		wantState pipeline.State
		wantURL   string
	}{
		"success": {runsJSON([3]string{completed, succeeded, "7"}), pipeline.StateSuccess, runURL("7")},
		"failure": {runsJSON([3]string{completed, failed, "7"}), pipeline.StateFailure, runURL("7")},
		"in progress": {
			runsJSON([3]string{inProgress, "", "8"}, [3]string{completed, succeeded, "7"}),
			pipeline.StateInProgress, runURL("8"),
		},
	}

	for caseName, stateCase := range stateCases {
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			fake := newFakeGitHub(t, answerWith(stateCase.body, nil))
			client, _ := newClient(t, fake)

			result, pollError := client.LatestRun(t.Context(), fake.query())
			if pollError != nil || result.Run.State != stateCase.wantState || result.Run.TargetURL != stateCase.wantURL {
				t.Fatalf("want %s at %s, got %+v (%v)", stateCase.wantState, stateCase.wantURL, result, pollError)
			}
		})
	}
}

func TestLatestRun_SendsTheDocumentedRequest(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, answerWith(runsJSON([3]string{completed, succeeded, "1"}), nil))
	client, _ := newClient(t, fake)

	_, pollError := client.LatestRun(t.Context(), fake.query())
	if pollError != nil {
		t.Fatal(pollError)
	}

	request := fake.lastRequest()

	checks := map[string][2]string{
		"path":          {request.URL.Path, runsPath},
		"branch":        {request.URL.Query().Get("branch"), "main"},
		"per page":      {request.URL.Query().Get("per_page"), "1"},
		"authorisation": {request.Header.Get("Authorization"), "Bearer " + fakeToken},
		"api version":   {request.Header.Get("X-Github-Api-Version"), "2022-11-28"},
		"accept":        {request.Header.Get("Accept"), "application/vnd.github+json"},
	}

	for checkName, gotAndWant := range checks {
		if gotAndWant[0] != gotAndWant[1] {
			t.Errorf("%s: want %q, got %q", checkName, gotAndWant[1], gotAndWant[0])
		}
	}
}

func TestLatestRun_SendsTheETagBack_AndReusesTheResultOnNotModified(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("If-None-Match") == `"etag-1"` {
			writer.WriteHeader(http.StatusNotModified)

			return
		}

		writer.Header().Set("ETag", `"etag-1"`)
		_, _ = writer.Write([]byte(runsJSON([3]string{completed, failed, "3"})))
	})
	client, counters := newClient(t, fake)

	first, firstError := client.LatestRun(t.Context(), fake.query())
	second, secondError := client.LatestRun(t.Context(), fake.query())

	if firstError != nil || secondError != nil {
		t.Fatalf("polling: %v %v", firstError, secondError)
	}

	if !second.FromCache || second.Run != first.Run {
		t.Fatalf("want the 304 answered from the cached run, got %+v", second)
	}

	snapshot := counters.Snapshot()
	if snapshot["pipeline.polls.not_modified"] != 1 || snapshot["pipeline.polls.ok"] != 1 {
		t.Fatalf("want one ok and one not modified poll counted, got %v", snapshot)
	}
}

func TestLatestRun_ReportsRateLimiting_WithTheWaitGitHubAskedFor(t *testing.T) {
	t.Parallel()

	rateCases := map[string]struct {
		status      int
		headers     map[string]string
		wantAtLeast time.Duration
	}{
		"429 with retry after": {
			http.StatusTooManyRequests, map[string]string{"Retry-After": "120"}, 120 * time.Second,
		},
		"403 with no remaining limit": {
			http.StatusForbidden, map[string]string{"X-Ratelimit-Remaining": "0"}, time.Minute,
		},
		"429 with nothing else": {http.StatusTooManyRequests, nil, time.Minute},
	}

	for caseName, rateCase := range rateCases {
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			fake := newFakeGitHub(t, func(writer http.ResponseWriter, _ *http.Request) {
				for name, value := range rateCase.headers {
					writer.Header().Set(name, value)
				}

				writer.WriteHeader(rateCase.status)
			})
			client, _ := newClient(t, fake)

			_, pollError := client.LatestRun(t.Context(), fake.query())

			failure, isFailure := errors.AsType[*pipeline.PollError](pollError)
			if !isFailure || failure.Category != "rate_limited" || failure.RetryAfter < rateCase.wantAtLeast {
				t.Fatalf("want rate_limited waiting at least %s, got %v", rateCase.wantAtLeast, pollError)
			}
		})
	}
}

func TestLatestRun_CategorisesEveryOtherFailure(t *testing.T) {
	t.Parallel()

	failureCases := map[string]struct {
		respond      func(http.ResponseWriter, *http.Request)
		wantCategory string
	}{
		"unauthorised": {answerStatus(http.StatusUnauthorized), "unauthorised"},
		"forbidden":    {answerStatus(http.StatusForbidden), "forbidden"},
		"not found":    {answerStatus(http.StatusNotFound), "not_found"},
		"server error": {answerStatus(http.StatusBadGateway), "server_error"},
		"malformed":    {answerWith(`{"workflow_runs": [`, nil), "malformed_response"},
		"oversized": {answerWith(`{"workflow_runs":[],"padding":"`+
			strings.Repeat("x", githubactions.MaximumResponseBytes)+`"}`, nil), "response_too_large"},
		"no runs": {answerWith(`{"total_count":0,"workflow_runs":[]}`, nil), "no_runs"},
	}

	for caseName, failureCase := range failureCases {
		t.Run(caseName, func(t *testing.T) {
			t.Parallel()

			fake := newFakeGitHub(t, failureCase.respond)
			client, _ := newClient(t, fake)

			_, pollError := client.LatestRun(t.Context(), fake.query())

			failure, isFailure := errors.AsType[*pipeline.PollError](pollError)
			if !isFailure || failure.Category != failureCase.wantCategory {
				t.Fatalf("want %s, got %v", failureCase.wantCategory, pollError)
			}

			if strings.Contains(pollError.Error(), fakeToken) {
				t.Fatal("the error must never contain the token")
			}
		})
	}
}

func TestLatestRun_RefusesQueriesBeforeAnyRequest(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, answerWith(runsJSON([3]string{completed, succeeded, "1"}), nil))
	client, _ := newClient(t, fake)

	badQueries := map[string]func(query *pipeline.Query){
		"plain http base":       func(query *pipeline.Query) { query.APIBaseURL = "http://api.github.com" },
		"repository with a dot": func(query *pipeline.Query) { query.Repository = "../../etc" },
		"workflow with a slash": func(query *pipeline.Query) { query.Workflow = "a/b" },
		"missing credential":    func(query *pipeline.Query) { query.CredentialName = "never-stored" },
	}

	for caseName, makeBad := range badQueries {
		query := fake.query()
		makeBad(&query)

		_, pollError := client.LatestRun(t.Context(), query)
		if _, isFailure := errors.AsType[*pipeline.PollError](pollError); !isFailure {
			t.Errorf("%s: want a categorised failure, got %v", caseName, pollError)
		}
	}

	if fake.requestCount() != 0 {
		t.Fatalf("want no request for a refused query, got %d", fake.requestCount())
	}
}

func TestLatestRun_HonoursTheDocumentedPollInterval(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, answerWith(runsJSON([3]string{completed, succeeded, "1"}),
		map[string]string{"X-Poll-Interval": "90", "X-Ratelimit-Remaining": "4321"}))
	client, counters := newClient(t, fake)

	result, pollError := client.LatestRun(t.Context(), fake.query())
	if pollError != nil || result.PollAfter != 90*time.Second {
		t.Fatalf("want a 90 second poll interval, got %+v (%v)", result, pollError)
	}

	if counters.Snapshot()["github.rate_limit_remaining"] != 4321 {
		t.Fatalf("want the remaining rate limit recorded, got %v", counters.Snapshot())
	}
}

// jobsJSON writes a run's jobs response, each job given as status and
// conclusion.
func jobsJSON(jobs ...[2]string) string {
	entries := make([]string, 0, len(jobs))
	for _, job := range jobs {
		conclusion := "null"
		if job[1] != "" {
			conclusion = `"` + job[1] + `"`
		}

		entries = append(entries, fmt.Sprintf(`{"status":"%s","conclusion":%s}`, job[0], conclusion))
	}

	return fmt.Sprintf(`{"total_count":%d,"jobs":[%s]}`, len(jobs), strings.Join(entries, ","))
}

// answerRunsAndJobs answers the runs endpoint with runs and any run's jobs
// endpoint with jobs.
func answerRunsAndJobs(runs string, jobs string) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		if strings.HasSuffix(request.URL.Path, "/jobs") {
			_, _ = writer.Write([]byte(jobs))

			return
		}

		_, _ = writer.Write([]byte(runs))
	}
}

// TestLatestRun_PressOpensTheNewestRun_EvenWhileItRuns: the key shows the
// newest run, so a press opens that run. Opening the last failure instead
// while a fix ran (TD10 part b) looked on the real deck like the key reporting
// the wrong run; the author amended TD10 (evaluation, deck testing).
func TestLatestRun_PressOpensTheNewestRun_EvenWhileItRuns(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, answerRunsAndJobs(
		runsJSON([3]string{inProgress, "", "9"}, [3]string{completed, failed, "8"}),
		jobsJSON([2]string{inProgress, ""}, [2]string{"queued", ""})))
	client, _ := newClient(t, fake)

	result, pollError := client.LatestRun(t.Context(), fake.query())
	if pollError != nil {
		t.Fatal(pollError)
	}

	if result.Run.State != pipeline.StateInProgress || result.Run.TargetURL != runURL("9") {
		t.Fatalf("want run 9 in progress and opened, got %+v", result.Run)
	}
}

// TestLatestRun_ShowsAFailedJob_BeforeTheRunFinishes: with one job failed and
// another still running, GitHub still calls the run in progress, which on the
// real deck hid a known failure for minutes (evaluation, deck testing). The
// author chose to show the failure at once.
func TestLatestRun_ShowsAFailedJob_BeforeTheRunFinishes(t *testing.T) {
	t.Parallel()

	for _, failedConclusion := range []string{failed, "timed_out"} {
		fake := newFakeGitHub(t, answerRunsAndJobs(
			runsJSON([3]string{inProgress, "", "9"}),
			jobsJSON([2]string{completed, succeeded}, [2]string{completed, failedConclusion},
				[2]string{inProgress, ""})))
		client, _ := newClient(t, fake)

		result, pollError := client.LatestRun(t.Context(), fake.query())
		if pollError != nil {
			t.Fatal(pollError)
		}

		if result.Run.State != pipeline.StateFailure || result.Run.TargetURL != runURL("9") {
			t.Errorf("%s: want run 9 shown as failed already, got %+v", failedConclusion, result.Run)
		}

		if path := fake.lastRequest().URL.Path; path != "/repos/octo-org/hello-world/actions/runs/9/jobs" {
			t.Errorf("want the run's jobs asked for, got %s", path)
		}
	}
}

// TestLatestRun_AsksForJobsOnlyWhileTheRunIsActive keeps the extra request to
// runs that have not finished: a finished run's own conclusion is final.
func TestLatestRun_AsksForJobsOnlyWhileTheRunIsActive(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, answerRunsAndJobs(
		runsJSON([3]string{completed, succeeded, "9"}), jobsJSON([2]string{completed, failed})))
	client, _ := newClient(t, fake)

	result, pollError := client.LatestRun(t.Context(), fake.query())
	if pollError != nil || result.Run.State != pipeline.StateSuccess {
		t.Fatalf("want run 9 succeeded, got %+v (%v)", result.Run, pollError)
	}

	if count := fake.requestCount(); count != 1 {
		t.Fatalf("want one request for a finished run, got %d", count)
	}
}

// TestLatestRun_SendsTheTokenOnlyToItsHost: the token goes only to the host it
// was stored for, api.github.com unless bound to another; a query naming any
// other host is refused before a request is made, whatever the configuration
// says.
func TestLatestRun_SendsTheTokenOnlyToItsHost(t *testing.T) {
	t.Parallel()

	fake := newFakeGitHub(t, answerWith(runsJSON([3]string{completed, succeeded, "1"}), nil))

	for name, binding := range map[string]string{"unbound": "", "bound elsewhere": "github.other.test"} {
		credentials := testsupport.NewCredentialStoreTestDouble()

		writeError := credentials.WriteCredential(fakeCredentialName, fakeToken)
		if writeError != nil {
			t.Fatal(writeError)
		}

		if binding != "" {
			bindError := platform.BindCredentialHost(credentials, fakeCredentialName, binding)
			if bindError != nil {
				t.Fatal(bindError)
			}
		}

		client := githubactions.NewClient(fake.server.Client(), credentials, diagnostics.NewCounters())

		_, pollError := client.LatestRun(t.Context(), fake.query())
		if reason := brick.ReasonCategoryOf(pollError); reason != "credential_host" {
			t.Errorf("%s: want the query refused as credential_host, got %q (%v)", name, reason, pollError)
		}
	}

	if fake.requestCount() != 0 {
		t.Fatalf("want no request, so no token sent, got %d", fake.requestCount())
	}
}
