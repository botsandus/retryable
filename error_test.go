package retryable

import "testing"

func TestMaxAttemptsReachedError(t *testing.T) {
	err := MaxAttemptsReachedError{c: 99}
	expect := "Request failed 99 times"

	if expect != err.Error() {
		t.Errorf("expected %q, received %q", expect, err.Error())
	}
}
