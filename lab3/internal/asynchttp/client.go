package asynchttp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"places/internal/future"
	"time"

	"github.com/jzx17/gofetch"
	"github.com/jzx17/gofetch/core"
)

const (
	userAgent        = "PlacesLab3/1.0 (educational web application)"
	jsonContentType  = "application/json"
	maxResponseBytes = 2 << 20
)

type Client struct{ transport *gofetch.Client }

// New создаёт HTTP-клиент с тайм-аутом и ограничением размера ответа.
func New(timeout time.Duration) *Client {
	return &Client{transport: gofetch.NewClient(
		gofetch.WithTimeout(timeout),
		gofetch.WithAutoBufferResponse(true),
		gofetch.WithSizeConfig(gofetch.SizeConfig{MaxResponseBodySize: maxResponseBytes}),
	)}
}

// GetJSON асинхронно загружает JSON и возвращает Future с разобранным ответом.
func GetJSON[T any](ctx context.Context, client *Client, endpoint string) *future.Future[T] {
	result := future.New[T]()
	requestURL, err := url.ParseRequestURI(endpoint)
	if err != nil || (requestURL.Scheme != "http" && requestURL.Scheme != "https") || requestURL.Host == "" {
		var zero T
		result.Resolve(zero, errors.New("invalid HTTP endpoint"))
		return result
	}
	responses := client.transport.GetAsync(ctx, endpoint, map[string]string{
		"User-Agent": userAgent, "Accept": jsonContentType,
	})
	// Ответ из канала ждёт отдельная горутина. Вызывающая функция сразу получает Future.
	go func() {
		var value T
		completion, ok := <-responses
		err := func() error {
			if !ok {
				return errors.New("HTTP client closed without a result")
			}
			if completion.Error != nil {
				err := sanitizeError(completion.Error)
				return fmt.Errorf("%s: request failed: %w", requestURL.Host, err)
			}
			response := completion.Response
			if response == nil {
				return errors.New("HTTP client returned no response")
			}
			defer func(response *core.Response) {
				err := response.CloseBody()
				if err != nil {
				}
			}(response)
			body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
			if err != nil {
				return err
			}
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("%s: HTTP %d", requestURL.Host, response.StatusCode)
			}
			if len(body) > maxResponseBytes {
				return fmt.Errorf("API response exceeds %d bytes", maxResponseBytes)
			}
			if err := json.Unmarshal(body, &value); err != nil {
				return fmt.Errorf("%s: invalid JSON: %w", requestURL.Host, err)
			}
			return nil
		}()
		result.Resolve(value, err)
	}()
	return result
}

// sanitizeError извлекает причину сетевой ошибки, исключая URL из сообщения.
func sanitizeError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
