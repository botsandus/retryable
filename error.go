package retryable

import "fmt"

// MaxAttemptsReachedError is returned, unsurprisingly, when we've attempted to upload
// data into the Digital Twin (ie: DexoryView) too many times, and none have been
// successful
type MaxAttemptsReachedError struct {
	c int
}

// Error implements the `Error` interface
func (e MaxAttemptsReachedError) Error() string {
	return fmt.Sprintf("Request failed %d times", e.c)
}
