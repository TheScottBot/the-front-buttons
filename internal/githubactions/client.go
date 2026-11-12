package githubactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/TheScottBot/the-front-buttons/internal/brick"
	"github.com/TheScottBot/the-front-buttons/internal/diagnostics"
	"github.com/TheScottBot/the-front-buttons/internal/pipeline"
	"github.com/TheScottBot/the-front-buttons/internal/platform"
)

// The request, as the GitHub REST documentation describes it (evaluation
// section 5, TD8).
const (
	apiVersion   = "2022-11-28"
	acceptHeader = "application/vnd.github+json"
	// runsPerPage asks for the newest run only: the key shows it and a press
	// opens it (TD10 as the author amended it, evaluation, deck testing).
	runsPerPage = 1
	// jobsPerPage covers every job of a run in one request; a matrix of two is
	// the usual case.
	jobsPerPage = 100
)

// MaximumResponseBytes bounds a response before it is parsed (datum 0.10).
// Twenty runs are well under a megabyte.
const MaximumResponseBytes = 1 << 20

// minimumRateLimitWait is GitHub's advice when rate limited with no other hint:
// "wait for at least one minute before retrying" (evaluation section 5).
const minimumRateLimitWait = time.Minute

// The reason categories a poll can fail with, shown on the key and counted.
const (
	categoryRefusedQuery      = "invalid_query"
	categoryCredentialMissing = "credential_missing" //nolint:gosec // a reason category's name, not a credential
	categoryCredentialError   = "credential_error"   //nolint:gosec // a reason category's name, not a credential
	categoryNetwork           = "network"
	categoryUnauthorised      = "unauthorised"
	categoryForbidden         = "forbidden"
	categoryRateLimited       = "rate_limited"
	categoryNotFound          = "not_found"
	categoryServerError       = "server_error"
	categoryUnexpectedStatus  = "unexpected_status"
	categoryTooLarge          = "response_too_large"
	categoryMalformed         = "malformed_response"
	categoryNoRuns            = "no_runs"
)

// The counter names this client keeps (specification section 7).
const (
	pollCounterPrefix     = "pipeline.polls."
	jobsCounterPrefix     = "pipeline.jobs."
	pollResultOK          = "ok"
	pollResultNotModified = "not_modified"
	rateLimitReading      = "github.rate_limit_remaining"
)

// The static errors behind each failure, wrapped with detail.
var (
	errQueryRefused     = errors.New("query refused")
	errHTTPStatus       = errors.New("unexpected HTTP status")
	errResponseTooLarge = errors.New("response too large")
)

// errNotModifiedWithoutCache reports a 304 for a request this client never
// cached, which a correct server cannot send.
var errNotModifiedWithoutCache = errors.New("not modified, but nothing was cached")

// cachedRun is the last full answer for one request, kept with its ETag so a
// 304 can be answered from it. 304s do not count against the rate limit when
// authorised (evaluation section 5).
type cachedRun struct {
	entityTag string
	run       pipeline.Run
	// activeRunID is the newest run's ID while it is queued or in progress,
	// whose jobs are then read for an early failure; zero once it finishes.
	activeRunID int64
}

// Client is the PipelineProvider adapter for GitHub Actions.
type Client struct {
	httpClient  *http.Client
	credentials platform.CredentialStore
	counters    *diagnostics.Counters
	now         func() time.Time

	mutex sync.Mutex
	cache map[string]cachedRun
}

// NewClient returns a client that reads tokens from the credential store.
func NewClient(httpClient *http.Client, credentials platform.CredentialStore, counters *diagnostics.Counters) *Client {
	return &Client{
		httpClient:  httpClient,
		credentials: credentials,
		counters:    counters,
		now:         time.Now,
		mutex:       sync.Mutex{},
		cache:       map[string]cachedRun{},
	}
}

// LatestRun returns a workflow's latest run on a branch, and the run a press
// opens.
func (client *Client) LatestRun(pollContext context.Context, query pipeline.Query) (pipeline.Result, error) {
	requestURL, queryError := runsURL(query)
	if queryError != nil {
		return client.fail(categoryRefusedQuery, 0, queryError)
	}

	token, credentialError := client.credentials.ReadCredential(query.CredentialName)

	switch {
	case errors.Is(credentialError, platform.ErrCredentialMissing):
		return client.fail(categoryCredentialMissing, 0, credentialError)
	case credentialError != nil:
		return client.fail(categoryCredentialError, 0, credentialError)
	}

	response, requestError := client.send(pollContext, requestURL, token)
	if requestError != nil {
		return client.fail(categoryNetwork, 0, requestError)
	}

	defer func() { _ = response.Body.Close() }()

	client.recordRateLimit(response.Header)

	result, interpretError := client.interpret(requestURL, response)
	if interpretError != nil {
		return result, interpretError
	}

	return client.withEarlyFailure(pollContext, query, requestURL, token, result), nil
}

// withEarlyFailure shows an active run as failed as soon as any of its jobs has
// failed. GitHub calls a run in progress until every job ends, which with a
// matrix hid a known failure for minutes on the real deck; the author chose to
// show it at once (evaluation, deck testing). If the jobs cannot be read, the
// run's own state stands.
func (client *Client) withEarlyFailure(
	pollContext context.Context,
	query pipeline.Query,
	requestURL string,
	token string,
	result pipeline.Result,
) pipeline.Result {
	client.mutex.Lock()
	activeRunID := client.cache[requestURL].activeRunID
	client.mutex.Unlock()

	if activeRunID > 0 && client.anyJobFailed(pollContext, query, activeRunID, token) {
		result.Run.State = pipeline.StateFailure
	}

	return result
}

// githubJobsResponse is the part of a run's jobs this client reads.
type githubJobsResponse struct {
	Jobs []struct {
		Conclusion *string `json:"conclusion"`
	} `json:"jobs"`
}

func (client *Client) anyJobFailed(pollContext context.Context, query pipeline.Query, runID int64, token string) bool {
	parameters := url.Values{}
	parameters.Set("per_page", strconv.Itoa(jobsPerPage))

	jobsURL := strings.TrimRight(query.APIBaseURL, "/") + "/repos/" + query.Repository +
		"/actions/runs/" + strconv.FormatInt(runID, 10) + "/jobs?" + parameters.Encode()

	response, requestError := client.send(pollContext, jobsURL, token)
	if requestError != nil {
		client.counters.Increment(jobsCounterPrefix + categoryNetwork)

		return false
	}

	defer func() { _ = response.Body.Close() }()

	client.recordRateLimit(response.Header)

	body, readError := io.ReadAll(io.LimitReader(response.Body, MaximumResponseBytes+1))

	var decoded githubJobsResponse

	if response.StatusCode != http.StatusOK || readError != nil || len(body) > MaximumResponseBytes ||
		json.Unmarshal(body, &decoded) != nil {
		client.counters.Increment(jobsCounterPrefix + categoryUnexpectedStatus)

		return false
	}

	client.counters.Increment(jobsCounterPrefix + pollResultOK)

	for _, job := range decoded.Jobs {
		if job.Conclusion != nil && StateOf(statusCompleted, *job.Conclusion) == pipeline.StateFailure {
			return true
		}
	}

	return false
}

func runsURL(query pipeline.Query) (string, error) {
	baseURL, parseError := url.Parse(query.APIBaseURL)

	switch {
	case parseError != nil || baseURL.Scheme != "https" || baseURL.Host == "" || baseURL.User != nil ||
		baseURL.RawQuery != "" || baseURL.Fragment != "":
		return "", fmt.Errorf("%w: the API base URL must be a plain https address (TD7)", errQueryRefused)
	// The same format checks the configuration applied, repeated here as a
	// guard so a value can never steer the request to another path.
	case !brick.CheckFormat(brick.FormatRepository, query.Repository):
		return "", fmt.Errorf("%w: the repository must be written owner/name", errQueryRefused)
	case !brick.CheckFormat(brick.FormatToken, query.Workflow):
		return "", fmt.Errorf("%w: the workflow must be a file name or numeric ID", errQueryRefused)
	case !brick.CheckFormat(brick.FormatBranch, query.Branch):
		return "", fmt.Errorf("%w: the branch name has characters a branch cannot", errQueryRefused)
	}

	parameters := url.Values{}
	parameters.Set("branch", query.Branch)
	parameters.Set("per_page", strconv.Itoa(runsPerPage))

	return strings.TrimRight(query.APIBaseURL, "/") + "/repos/" + query.Repository +
		"/actions/workflows/" + url.PathEscape(query.Workflow) + "/runs?" + parameters.Encode(), nil
}

func (client *Client) send(pollContext context.Context, requestURL string, token string) (*http.Response, error) {
	request, buildError := http.NewRequestWithContext(pollContext, http.MethodGet, requestURL, http.NoBody)
	if buildError != nil {
		return nil, fmt.Errorf("building the request: %w", buildError)
	}

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", acceptHeader)
	request.Header.Set("X-Github-Api-Version", apiVersion)

	client.mutex.Lock()
	cached, isCached := client.cache[requestURL]
	client.mutex.Unlock()

	if isCached && cached.entityTag != "" {
		request.Header.Set("If-None-Match", cached.entityTag)
	}

	response, sendError := client.httpClient.Do(request)
	if sendError != nil {
		return nil, fmt.Errorf("requesting workflow runs: %w", sendError)
	}

	return response, nil
}

func (client *Client) interpret(requestURL string, response *http.Response) (pipeline.Result, error) {
	switch {
	case response.StatusCode == http.StatusOK:
		return client.readRuns(requestURL, response)
	case response.StatusCode == http.StatusNotModified:
		return client.fromCache(requestURL, response)
	case isRateLimited(response):
		return client.fail(categoryRateLimited, client.rateLimitWait(response.Header),
			fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode))
	case response.StatusCode == http.StatusUnauthorized:
		return client.fail(categoryUnauthorised, 0, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode))
	case response.StatusCode == http.StatusForbidden:
		return client.fail(categoryForbidden, 0, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode))
	case response.StatusCode == http.StatusNotFound:
		return client.fail(categoryNotFound, 0, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode))
	case response.StatusCode >= http.StatusInternalServerError:
		return client.fail(categoryServerError, 0, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode))
	default:
		return client.fail(categoryUnexpectedStatus, 0, fmt.Errorf("%w: %d", errHTTPStatus, response.StatusCode))
	}
}

// isRateLimited applies GitHub's description: a 429, or a 403 with no remaining
// primary limit or with a retry after header (evaluation section 5).
func isRateLimited(response *http.Response) bool {
	if response.StatusCode == http.StatusTooManyRequests {
		return true
	}

	return response.StatusCode == http.StatusForbidden &&
		(response.Header.Get("X-Ratelimit-Remaining") == "0" || response.Header.Get("Retry-After") != "")
}

// rateLimitWait is how long GitHub asked to wait: retry after if given, else
// until the reset if the limit is spent, and never less than a minute.
func (client *Client) rateLimitWait(header http.Header) time.Duration {
	wait := minimumRateLimitWait

	retryAfter, retryParseError := strconv.Atoi(header.Get("Retry-After"))
	if retryParseError == nil {
		wait = max(wait, time.Duration(retryAfter)*time.Second)
	}

	resetSeconds, resetParseError := strconv.ParseInt(header.Get("X-Ratelimit-Reset"), 10, 64)
	if header.Get("X-Ratelimit-Remaining") == "0" && resetParseError == nil {
		wait = max(wait, time.Unix(resetSeconds, 0).Sub(client.now()))
	}

	return wait
}

// githubRunsResponse is the part of the response this client reads. Other
// fields are ignored rather than refused (deviation DV4): the response is
// GitHub's, which adds fields over time, not a contract this project owns. The
// field names are GitHub's own, in its snake case.
//
//nolint:tagliatelle // the names are GitHub's, not this project's
type githubRunsResponse struct {
	WorkflowRuns []struct {
		ID         int64   `json:"id"`
		Status     string  `json:"status"`
		Conclusion *string `json:"conclusion"`
		HTMLURL    string  `json:"html_url"`
	} `json:"workflow_runs"`
}

func (client *Client) readRuns(requestURL string, response *http.Response) (pipeline.Result, error) {
	body, readError := io.ReadAll(io.LimitReader(response.Body, MaximumResponseBytes+1))

	switch {
	case readError != nil:
		return client.fail(categoryNetwork, 0, readError)
	case len(body) > MaximumResponseBytes:
		return client.fail(categoryTooLarge, 0, fmt.Errorf("%w: over %d bytes", errResponseTooLarge, MaximumResponseBytes))
	}

	var decoded githubRunsResponse

	decodeError := json.Unmarshal(body, &decoded)
	if decodeError != nil {
		return client.fail(categoryMalformed, 0, decodeError)
	}

	if len(decoded.WorkflowRuns) == 0 {
		return client.fail(categoryNoRuns, 0, nil)
	}

	newest := decoded.WorkflowRuns[0]

	conclusion := ""
	if newest.Conclusion != nil {
		conclusion = *newest.Conclusion
	}

	run := pipeline.Run{State: StateOf(newest.Status, conclusion), TargetURL: httpsOnly(newest.HTMLURL)}

	activeRunID := int64(0)
	if newest.Status != statusCompleted && newest.ID > 0 {
		activeRunID = newest.ID
	}

	client.mutex.Lock()
	client.cache[requestURL] = cachedRun{entityTag: response.Header.Get("ETag"), run: run, activeRunID: activeRunID}
	client.mutex.Unlock()

	client.counters.Increment(pollCounterPrefix + pollResultOK)

	return pipeline.Result{Run: run, PollAfter: pollInterval(response.Header), FromCache: false}, nil
}

func (client *Client) fromCache(requestURL string, response *http.Response) (pipeline.Result, error) {
	client.mutex.Lock()
	cached, isCached := client.cache[requestURL]
	client.mutex.Unlock()

	if !isCached {
		return client.fail(categoryMalformed, 0, errNotModifiedWithoutCache)
	}

	client.counters.Increment(pollCounterPrefix + pollResultNotModified)

	return pipeline.Result{Run: cached.run, PollAfter: pollInterval(response.Header), FromCache: true}, nil
}

// httpsOnly keeps a run's address only if it is https. The address comes from
// the response, so it is untrusted; the key opens it, so nothing else is let
// through.
func httpsOnly(address string) string {
	parsed, parseError := url.Parse(address)
	if parseError != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}

	return address
}

// pollInterval reads the x-poll-interval header, which GitHub asks clients to
// obey (TD8).
func pollInterval(header http.Header) time.Duration {
	seconds, parseError := strconv.Atoi(header.Get("X-Poll-Interval"))
	if parseError != nil || seconds <= 0 {
		return 0
	}

	return time.Duration(seconds) * time.Second
}

func (client *Client) recordRateLimit(header http.Header) {
	remaining, parseError := strconv.ParseInt(header.Get("X-Ratelimit-Remaining"), 10, 64)
	if parseError == nil {
		client.counters.Set(rateLimitReading, remaining)
	}
}

func (client *Client) fail(category string, retryAfter time.Duration, cause error) (pipeline.Result, error) {
	client.counters.Increment(pollCounterPrefix + category)

	return pipeline.Result{Run: pipeline.Run{State: "", TargetURL: ""}, PollAfter: 0, FromCache: false},
		&pipeline.PollError{Category: category, RetryAfter: retryAfter, Cause: cause}
}
