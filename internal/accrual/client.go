package accrual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	StatusRegistered = "REGISTERED"
	StatusProcessing = "PROCESSING"
	StatusInvalid    = "INVALID"
	StatusProcessed  = "PROCESSED"
)

var (
	ErrOrderNotRegistered = errors.New("order not registered")
	ErrUnexpectedResponse = errors.New("unexpected accrual response")
)

type RateLimitError struct {
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("accrual rate limit: retry after %s", e.RetryAfter)
}

type Order struct {
	Number  string   `json:"order"`
	Status  string   `json:"status"`
	Accrual *float64 `json:"accrual,omitempty"`
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}

	parsedBaseURL, err := parseBaseURL(baseURL)
	if err != nil {
		parsedBaseURL = nil
	}

	return &Client{
		baseURL:    parsedBaseURL,
		httpClient: httpClient,
	}
}

func (c *Client) GetOrder(ctx context.Context, number string) (Order, error) {
	if c.baseURL == nil {
		return Order{}, errors.New("accrual base URL is invalid")
	}

	requestURL := *c.baseURL
	requestURL.Path = strings.TrimRight(requestURL.Path, "/") + "/api/orders/" + url.PathEscape(number)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		requestURL.String(),
		nil,
	)
	if err != nil {
		return Order{}, fmt.Errorf("create accrual request: %w", err)
	}

	//  Валидируем урлу через parseBaseURL, а путь фиксируем на уровне контракта, так что игнорим ворнинг
	resp, err := c.httpClient.Do(req) //nolint:gosec
	if err != nil {
		return Order{}, fmt.Errorf("send accrual request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var order Order
		if decodeErr := json.NewDecoder(resp.Body).Decode(&order); decodeErr != nil {
			return Order{}, fmt.Errorf("decode accrual response: %w", decodeErr)
		}
		return order, nil
	case http.StatusNoContent:
		return Order{}, ErrOrderNotRegistered
	case http.StatusTooManyRequests:
		retryAfter, parseErr := parseRetryAfter(resp.Header.Get("Retry-After"))
		if parseErr != nil {
			return Order{}, parseErr
		}
		return Order{}, &RateLimitError{RetryAfter: retryAfter}
	default:
		return Order{}, fmt.Errorf("%w: status %d", ErrUnexpectedResponse, resp.StatusCode)
	}
}

func parseRetryAfter(value string) (time.Duration, error) {
	seconds, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || seconds <= 0 {
		return 0, fmt.Errorf("invalid Retry-After header %q", value)
	}

	return time.Duration(seconds) * time.Second, nil
}

func parseBaseURL(raw string) (*url.URL, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, errors.New("empty base URL")
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported base URL scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return nil, errors.New("base URL host is empty")
	}
	if parsed.User != nil {
		return nil, errors.New("base URL must not contain user info")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("base URL must not contain query or fragment")
	}

	return parsed, nil
}
