// Package asynchttp adapts net/http to an asynchronous Future-based interface.
package asynchttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"places/internal/future"
)

const (
	userAgent        = "PlacesLab3/1.0 (educational console application)"
	jsonContentType  = "application/json"
	maxResponseBytes = 2 << 20
)

type Client struct{ HTTP *http.Client }

// GetJSON returns immediately; transport and decoding run in a worker goroutine.
func GetJSON[T any](ctx context.Context, client *Client, endpoint string) *future.Future[T] {
	result := future.New[T]()
	go func() {
		var value T
		err := func() error {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err != nil {
				return err
			}
			request.Header.Set("User-Agent", userAgent)
			request.Header.Set("Accept", jsonContentType)
			response, err := client.HTTP.Do(request)
			if err != nil {
				return err
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {

				}
			}(response.Body)
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("%s: HTTP %d", request.URL.Host, response.StatusCode)
			}
			body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
			if err != nil {
				return err
			}
			if len(body) > maxResponseBytes {
				return fmt.Errorf("API response exceeds %d bytes", maxResponseBytes)
			}
			if err := json.Unmarshal(body, &value); err != nil {
				return fmt.Errorf("%s: invalid JSON: %w", request.URL.Host, err)
			}
			return nil
		}()
		result.Resolve(value, err)
	}()
	return result
}
