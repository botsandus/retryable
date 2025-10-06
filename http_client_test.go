package retryable_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/botsandus/retryable"
)

func TestNew(t *testing.T) {
	c := retryable.New()
	if c == nil {
		t.Fatal("value must not be nil")
	}
}

func TestHttpClient_DoWithContext(t *testing.T) {
	for _, test := range []struct {
		name           string
		resp           int
		expectAttempts int
		expectDuration bool
		expectError    bool
	}{
		{"Redirect loop fails early", http.StatusTemporaryRedirect, 1, false, true},
		{"Rate limited calls will wait", http.StatusTooManyRequests, 2, false, true},
		{"404s fail early", http.StatusNotFound, 1, false, true},
		{"500s keep retrying", http.StatusInternalServerError, 2, false, true},
		{"200s return immediately with timing", http.StatusOK, 1, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("Retry-After", "1")    // only used with status 429
				w.Header().Add("Location", "/a/a/a/") // only used with status 307
				w.WriteHeader(test.resp)
			}))
			defer ts.Close()

			req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			c := retryable.New()
			c.MaxInterval = time.Millisecond
			c.MaxRetries = 1

			ctx := retryable.NewContext()

			_, err = c.DoWithContext(ctx, req)
			if test.expectError == (err == nil) {
				t.Errorf("expected: %v, received %#v", test.expectError, err)
			}

			t.Run("attempts count", func(t *testing.T) {
				attempts, ok := retryable.NumberOfAttemptsFromContext(ctx)
				if !ok {
					t.Fatal("expected `attempts` in the context")
				}

				if test.expectAttempts != attempts {
					t.Errorf("expected %d, received %d", test.expectAttempts, attempts)
				}
			})

			t.Run("successful duration", func(t *testing.T) {
				dur, ok := retryable.SuccessfulRequestDurationFromContext(ctx)

				if !ok {
					t.Fatal("expected `duration` in the context")
				}

				if test.expectDuration == (dur.Nanoseconds() == 0) {
					t.Errorf("expectedDuration is %v, yet duration was %d ns", test.expectDuration, dur.Milliseconds())
				}
			})

		})
	}
}

func TestHttpClient_DoWithContext_No429ReryAfter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := retryable.New()

	ctx := context.Background()

	_, err = c.DoWithContext(ctx, req)
	if err == nil {
		t.Error("request should have failed")
	}

	_, ok := retryable.NumberOfAttemptsFromContext(ctx)
	if ok {
		t.Error("no attempts should have been returned")
	}
}

// TestHttpClient_DoWithContext_NakedContexts tests the HttpClient doesn't fall over
// if we use a naked context.Context from the standard library, in scenarios where
// we may not care about metadata for metrics
func TestHttpClient_DoWithContext_NakedContexts(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Retry-After", "1")    // only used with status 429
		w.Header().Add("Location", "/a/a/a/") // only used with status 307
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := retryable.New()

	ctx := context.Background()

	_, err = c.DoWithContext(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := retryable.NumberOfAttemptsFromContext(ctx)
	if ok {
		t.Error("no attempts should have been returned")
	}

	_, ok = retryable.SuccessfulRequestDurationFromContext(ctx)
	if ok {
		t.Error("no duration should have been returned")
	}
}

// TestHttpClient_DoWithContext_UseMaxElapsedTime tests that MaxRetries=0 respects MaxElapsedTime timeout
func TestHttpClient_DoWithContext_UseMaxElapsedTime(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	c := retryable.New()
	c.MaxRetries = 0                       // Set MaxRetries to 0
	c.MaxElapsedTime = 1 * time.Second     // Allow 1 second for retries
	c.MaxInterval = 100 * time.Millisecond // Short intervals

	ctx := retryable.NewContext()
	start := time.Now()

	_, err = c.DoWithContext(ctx, req)
	elapsed := time.Since(start)

	// Should fail due to MaxElapsedTime being exceeded
	if err == nil {
		t.Error("expected request to fail due to MaxElapsedTime exceeded")
	}

	// Should have taken approximately MaxElapsedTime
	if elapsed < 800*time.Millisecond || elapsed > 1500*time.Millisecond {
		t.Errorf("expected elapsed time around 1s, got %v", elapsed)
	}

	attempts, ok := retryable.NumberOfAttemptsFromContext(ctx)
	if !ok {
		t.Fatal("expected attempts in the context")
	}

	if attempts < 2 {
		t.Errorf("expected at least 2 attempts with MaxRetries=0 and MaxElapsedTime, got %d", attempts)
	}
}

func TestHttpClient_DoWithContext_WithHomegrownRequest(t *testing.T) {
	var (
		size  int
		calls int
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		if calls == 5 {
			buf := new(bytes.Buffer)
			io.Copy(buf, r.Body)
			r.Body.Close()

			size = buf.Len()

			w.WriteHeader(http.StatusOK)

			return
		}

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	payload := `{"msg":"hello, world!"}`

	req, err := retryable.NewRequest(http.MethodPost, ts.URL, bytes.NewReader([]byte(payload)))
	if err != nil {
		t.Fatal(err)
	}

	c := retryable.New()
	c.MaxRetries = 0                      // Set MaxRetries to 0
	c.MaxElapsedTime = 1 * time.Second    // Allow 1 second for retries
	c.MaxInterval = 10 * time.Millisecond // Short intervals

	_, err = c.DoWithContext(context.Background(), req)
	if err != nil {
		t.Error(err)
	}

	if calls != 5 {
		t.Errorf("expected 5 requests, received %d", calls)
	}

	if size != len(payload) {
		t.Errorf("expected a payload of %d bytes, received %d bytes", len(payload), size)
	}
}
