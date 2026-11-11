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

	"github.com/jzx17/gofetch"
)

const (
	userAgent        = "PlacesLab3/1.0 (educational web application)"
	jsonContentType  = "application/json"
	maxResponseBytes = 2 << 20
	maxLogBodyBytes  = 4096
)

var requestSequence atomic.Uint64

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
	id := requestSequence.Add(1)
	started := time.Now()
	safeURL, redact := logURL(requestURL)
	log.Printf("[API #%d] GET %s", id, safeURL)
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
				log.Printf("[API #%d] FAILED after %s: %s", id, time.Since(started).Round(time.Millisecond), redact(err.Error()))
				return fmt.Errorf("%s: request failed: %w", requestURL.Host, err)
			}
			response := completion.Response
			if response == nil {
				return errors.New("HTTP client returned no response")
			}
			defer response.CloseBody()
			body, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
			bodyLog := redact(string(body))
			if len(bodyLog) > maxLogBodyBytes {
				bodyLog = bodyLog[:maxLogBodyBytes] + " ... [truncated]"
			}
			log.Printf("[API #%d] %s HTTP %d in %s; body=%q", id, requestURL.Host, response.StatusCode, time.Since(started).Round(time.Millisecond), bodyLog)
			if err != nil {
				log.Printf("[API #%d] READ FAILED: %s", id, redact(err.Error()))
				return err
			}
			if response.StatusCode != http.StatusOK {
				return fmt.Errorf("%s: HTTP %d", requestURL.Host, response.StatusCode)
			}
			if len(body) > maxResponseBytes {
				log.Printf("[API #%d] RESPONSE TOO LARGE", id)
				return fmt.Errorf("API response exceeds %d bytes", maxResponseBytes)
			}
			if err := json.Unmarshal(body, &value); err != nil {
				log.Printf("[API #%d] INVALID JSON: %s", id, redact(err.Error()))
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

// logURL маскирует ключи в адресе и готовит функцию очистки текста для журнала.
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
