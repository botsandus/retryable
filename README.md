# retryable

[![Go Reference](https://pkg.go.dev/badge/github.com/botsandus/retryable.svg)](https://pkg.go.dev/github.com/botsandus/retryable)

Retryable is an HTTP Client, based on the bog-standard `net/http` client we all know and love, with an exponential backoff, rate-limit support, and which exposes some sensible numbers which can be used to plug into various places.


## Usage

```golang
package main

import (
    "fmt"
    "net/http"

    "github.com/botsandus/retryable"
)

func main() {
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
```

## Configuration

The client will initialise with some sensible defaults which you can find [here](https://github.com/botsandus/retryable/blob/main/http_client.go) as part of the `HttpClient`.

You can override these like so:

```
 c := retryable.New()
 c.MaxRetries = 99                   // Will try a total of 100 times.
 c.MaxInterval = 5 time.Minute       // Intervals between retries shouldn't exceed 5 minutes.
 c.MaxElapsedTime = 10 * time.Minute // Stops retrying completely after 10 minutes.
```
