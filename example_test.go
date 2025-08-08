package retryable_test

import (
	"fmt"
	"net/http"

	"github.com/botsandus/retryable"
)

func ExampleHttpClient_DoWithContext() {
	req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
	if err != nil {
		panic(err)
	}

	c := retryable.New()
	ctx := retryable.NewContext()

	resp, err := c.DoWithContext(ctx, req)
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Status)

	attempts, ok := retryable.NumberOfAttemptsFromContext(ctx)
	if !ok {
		fmt.Println("unable to get request count")
	}

	fmt.Printf("It took %d attempts to successfully make this call", attempts)

	duration, ok := retryable.SuccessfulRequestDurationFromContext(ctx)
	if !ok {
		fmt.Println("unable to get request count")
	}

	fmt.Printf("The successful attempt ran with a duration of %s", duration)
}
