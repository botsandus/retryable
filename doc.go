/*
Package retryable provides exponential backoff, `429 Too Many Requests`, and metrics support around
the basic `net/http` client in the standard library.

It is designed to be an _almost_ API compatible wrapper by wrapping the default client, and adding the
function `DoWithContext` - which is identical to `Do`, with the addition of a context.
*/
package retryable
