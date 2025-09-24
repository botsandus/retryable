package retryable

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	backoff "github.com/cenkalti/backoff/v5"
)

var (
	// The following errors strings are used to determine whether an http
	// request has failed in an exciting way. The net/http package doesn't
	// use specific error types that we can plug into `errors.Is(err, ..)`,
	// nor does it export these strings. In actual fact, net/http returns errors
	// as magic strings, wrapped in `errors.New(..)` and then further wrapped with
	// request context- so we can't even use string equality checking.
	//
	// Thanks Rob Pike
	redirectErrorString      = regexp.MustCompile("stopped after 10 redirects")
	untrustedCertErrorString = regexp.MustCompile("certificate is not trusted")

	// default429RetrySeconds is used in the case of 429s that don't set the Retry-After
	// header, which only _may_ be included according to rfc6585.
	//
	// To understand the semantics of the word _may_, please see rfc2119
	default429RetrySeconds = 1
)

// An HttpClient wraps the default net/http client with a backoff function,
// allowing for transient failures to be retried
type HttpClient struct {
	*http.Client

	MaxRetries     int
	MaxInterval    time.Duration
	MaxElapsedTime time.Duration
}

// New returns an HttpClient with some retry logic attached
func New() *HttpClient {
	return &HttpClient{
		MaxRetries:     9, // For a total of 10 calls, by default
		MaxInterval:    time.Second * 30,
		MaxElapsedTime: 0, // Never gonna give you up
		Client:         http.DefaultClient,
	}
}

// DoWithContext wraps the http.Client.Do function, accepting an additional context
// which can be used to return metadata about this call, including request attempts,
// durations, and so on.
//
// DoWithContext will return an early error if the url in the request has an error, if
// the server redirects too many times, if the server has a dodgy cert, or if the server
// returns a non-429 4xx error.
//
// Anything else is retried.
func (h HttpClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	// Create a backoff per request; they're not thread safe
	bo := backoff.NewExponentialBackOff()
	bo.MaxInterval = h.MaxInterval

	metadata, ok := getRequestMetadata(ctx)
	if !ok {
		// If we get a context not created by HttpClient.NewContext() then that's
		// cool, we just wont be able to do anything with it
		metadata = new(requestMetadata)
	}

	metadata.requests = 0

	operation := func() (*http.Response, error) {
		metadata.requests++

		// If we've used up all of our request attempts, return so we can
		// log accordingly.
		//
		// Note that we add `1` to the number of MaxRetries since the first
		// attempt isn't a retry, it's a _try_
		if metadata.requests >= h.MaxRetries+1 {
			return nil, &backoff.PermanentError{
				Err: MaxAttemptsReachedError{c: h.MaxRetries + 1},
			}
		}

		start := time.Now()
		resp, err := h.Do(req)
		requestDuration := time.Since(start)

		if err != nil {
			switch {
			case redirectErrorString.MatchString(err.Error()),
				untrustedCertErrorString.MatchString(err.Error()):
				return nil, backoff.Permanent(err)
			}

			// Any further error may be transient and, as such, is
			// retryable
			return nil, err
		}

		// If we are being rate limited, return a RetryAfter to specify how long to wait.
		// This will also reset the backoff policy.
		if resp.StatusCode == 429 {
			ra := resp.Header.Get("Retry-After")
			if ra == "" {
				return nil, backoff.RetryAfter(default429RetrySeconds)
			}

			seconds, err := strconv.ParseInt(ra, 10, 64)
			if err != nil {
				return nil, err
			}

			return nil, backoff.RetryAfter(int(seconds))
		}

		// Treat any non 429 client error as a permanent error
		if resp.StatusCode/100 == 4 {
			return resp, backoff.Permanent(errors.New(resp.Status))
		}

		// Treat any other non-2xx status as a transient error (the DefaultClient from
		// `net/http` already handles 3xx redirects, so we're in no danger of breaking
		// those here)
		if resp.StatusCode/100 != 2 {
			return resp, errors.New(resp.Status)
		}

		// If we get this far, the operation succeeded; update the duration, and return
		metadata.successfulDuration = requestDuration

		return resp, nil
	}

	return backoff.Retry(ctx, operation, backoff.WithBackOff(bo), backoff.WithMaxElapsedTime(h.MaxElapsedTime))
}
