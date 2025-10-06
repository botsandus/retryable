package retryable

import (
	"bytes"
	"io"
	"net/http"
)

// NewRequestWithBodyBytes creates an HTTP request with a body that can be retried.
// The body must be provided as []byte so it can be rewound between retry attempts.
func NewRequestWithBodyBytes(method, url string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// This will get called as part of each retry.
	// It basically adds some resilience in case we get network interrupts,
	// to ensure the full body gets sent for each retry.
	// Otherwise, we may send a partial upload which the server might reject
	// as the content length won't match the header.
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}

	return req, nil
}
