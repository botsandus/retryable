package retryable

import (
	"context"
	"time"
)

// requestMetadata is stored as a pointer inside our contexts to allow us to
// pass metadata around
type requestMetadata struct {
	requests           int
	successfulDuration time.Duration
}

// httpRequestMetadataContextKey is used to key metadata within request contexts
type httpRequestMetadataContextKey struct{}

// NewContext returns a context.Context preseeded for DoWithContext use,
// with handy things such as metadata keys pre-created
func NewContext() context.Context {
	return context.WithValue(context.Background(), httpRequestMetadataContextKey{}, new(requestMetadata))
}

func getRequestMetadata(ctx context.Context) (*requestMetadata, bool) {
	v := ctx.Value(httpRequestMetadataContextKey{})

	ptr, ok := v.(*requestMetadata)

	return ptr, ok
}

// NumberOfAttemptsFromContext may be used to return the number of attempts the httpClient
// took in order to get a successful response
func NumberOfAttemptsFromContext(ctx context.Context) (int, bool) {
	md, ok := getRequestMetadata(ctx)
	if !ok {
		return 0, false
	}

	return md.requests, true
}

// SuccessfulRequestDurationFromContext may be used to return the duration the upload to
// DexoryView took, should there have been a successful request
func SuccessfulRequestDurationFromContext(ctx context.Context) (time.Duration, bool) {
	md, ok := getRequestMetadata(ctx)
	if !ok {
		return 0, false
	}

	return md.successfulDuration, true
}
