package asynchttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"places/internal/future"
	"strings"
	"sync/atomic"
	"time"
)

const (
	userAgent        = "PlacesLab3/1.0 (educational web application)"
	jsonContentType  = "application/json"
	maxResponseBytes = 2 << 20
	maxLogBodyBytes  = 4096
)

var requestSequence atomic.Uint64

type Client struct{ HTTP *http.Client }

func GetJSON[T any](ctx context.Context, client *Client, endpoint string) *future.Future[T] {
	result := future.New[T]()
	go func() {
		var value T
		err := func() error {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
			if err != nil {
				return err
			}
			id := requestSequence.Add(1)
			started := time.Now()
			safeURL, redact := logURL(request.URL)
			log.Printf("[API #%d] GET %s", id, safeURL)
			request.Header.Set("User-Agent", userAgent)
			request.Header.Set("Accept", jsonContentType)
			response, err := client.HTTP.Do(request)
			if err != nil {
				log.Printf("[API #%d] FAILED after %s: %s", id, time.Since(started).Round(time.Millisecond), redact(sanitizeError(err).Error()))
				return fmt.Errorf("%s: request failed: %w", request.URL.Host, sanitizeError(err))
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
				}
			}(response.Body)
			body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
			bodyLog := redact(string(body))
			if len(bodyLog) > maxLogBodyBytes {
				bodyLog = bodyLog[:maxLogBodyBytes] + " ... [truncated]"
			}
			log.Printf("[API #%d] %s HTTP %d in %s; body=%q", id, request.URL.Host, response.StatusCode, time.Since(started).Round(time.Millisecond), bodyLog)
			if err != nil {
				log.Printf("[API #%d] READ FAILED: %s", id, redact(err.Error()))
				return err
			}
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("%s: HTTP %d", request.URL.Host, response.StatusCode)
			}
			if len(body) > maxResponseBytes {
				log.Printf("[API #%d] RESPONSE TOO LARGE", id)
				return fmt.Errorf("API response exceeds %d bytes", maxResponseBytes)
			}
			if err := json.Unmarshal(body, &value); err != nil {
				log.Printf("[API #%d] INVALID JSON: %s", id, redact(err.Error()))
				return fmt.Errorf("%s: invalid JSON: %w", request.URL.Host, err)
			}
			return nil
		}()
		result.Resolve(value, err)
	}()
	return result
}

func sanitizeError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}

func logURL(source *url.URL) (string, func(string) string) {
	safe := *source
	query := safe.Query()
	var secrets []string
	for key, values := range query {
		switch strings.ToLower(key) {
		case "key", "appid", "api_key", "apikey", "token":
			for _, value := range values {
				if value != "" {
					secrets = append(secrets, value)
				}
			}
			query.Set(key, "[REDACTED]")
		}
	}
	safe.User = nil
	safe.RawQuery = query.Encode()
	return safe.String(), func(text string) string {
		for _, secret := range secrets {
			encoded, _ := json.Marshal(secret)
			for _, variant := range []string{string(encoded[1 : len(encoded)-1]), url.QueryEscape(secret), secret} {
				text = strings.ReplaceAll(text, variant, "[REDACTED]")
			}
		}
		return text
	}
}
