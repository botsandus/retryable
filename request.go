package retryable

import (
	"bytes"
	"io"
	"net/http"
)

// NewRequest wraps the function from net/http, but with the addition
// of a `GetBody` function on that request.
//
// This will allow a developer to be safe in the knowledge that all requests
// made, on all attempts, will always have the correct payload.
//
// This is because there's a special case in `net/http` whereby a request which
// fails part way through one attempt will only attempt to send some data when
// retried. Somewhere in the bowels of Go, the request will check for the existence
// of a `Body` on a request and, if it's set, will read from it. If that read returns
// an io.EOF immediately, then Go will try to rewind it.
//
// If a request has a Body of 100mb, and that request fails 70mb into an upload,
// the retry will only upload the last 30mb- which is probably broken.
//
// Note: you're probably better off providing your own `req.GetBody` function; especially
// on large requests- this function will read your body into memory, persisting a copy
// of it until the request finally succeeds and the copy is garbage collected.
func NewRequest(method, url string, body io.Reader) (*http.Request, error) {
	buf := new(bytes.Buffer)

	_, err := io.Copy(buf, body)
	if err != nil {
		return nil, err
	}

	bb := buf.Bytes()

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		return nil, err
	}

	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(bb)), nil
	}

	return req, nil
}
