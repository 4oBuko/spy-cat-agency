package catapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type CatAPI interface {
	GetBreedById(ctx context.Context, id string) (Breed, error)
}

type CatAPIClient struct {
	url        string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
	breeds     []Breed
}

var ErrBreedNotFound = errors.New("breed not found")

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func NewCatAPIClient(url string, maxRetry int, retryDelay time.Duration) *CatAPIClient {
	return &CatAPIClient{
		url: url,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		maxRetries: 1,
		retryDelay: 1 * time.Second,
	}
}
func (c *CatAPIClient) getAllBreeds(ctx context.Context) ([]Breed, error) {
	if c.breeds == nil {
		c.fetchAllBreeds(ctx)
	}
	return c.breeds, nil
}

func (c *CatAPIClient) GetBreedById(ctx context.Context, id string) (Breed, error) {
	breeds, err := c.getAllBreeds(ctx)
	if err != nil {
		return Breed{}, fmt.Errorf("error while fetching breeds: %w", err)
	}
	for _, breed := range breeds {
		if breed.Id == id {
			return breed, nil
		}
	}
	return Breed{}, ErrBreedNotFound
}

func (c *CatAPIClient) fetchAllBreeds(ctx context.Context) error {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)): // Exponential backoff
			}
		}

		breeds, err := c.makeGetAllBreedsRequest(ctx)
		if err == nil {
			c.breeds = breeds
			return nil
		}

		lastErr = err

		if httpErr, ok := err.(*HTTPError); ok {
			if !c.isRetryableError(nil, httpErr.StatusCode) {
				break
			}
		}

		fmt.Printf("Attempt %d failed: %v, retrying...\n", attempt+1, err)
	}

	return fmt.Errorf("failed after %d attempts, last error: %w", c.maxRetries+1, lastErr)
}

func (c *CatAPIClient) makeGetAllBreedsRequest(ctx context.Context) ([]Breed, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to the api failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, &HTTPError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", response.StatusCode),
		}
	}

	var breeds []Breed
	if err := json.NewDecoder(response.Body).Decode(&breeds); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return breeds, nil

}

func (c *CatAPIClient) isRetryableError(err error, statusCode int) bool {
	if err != nil {
		return true // Network errors are retryable
	}

	return statusCode >= 500 || statusCode == http.StatusRequestTimeout || statusCode == http.StatusTooManyRequests
}
