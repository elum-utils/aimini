package aimini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Health calls GET /health.
func (c *Client) Health(ctx context.Context) (*HealthResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/health"), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}
	defer resp.Body.Close()

	var out HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("aimini: decode health response: %w", err)
	}
	return &out, nil
}

// DownloadCDN downloads a CDN object returned as input_s3_url or output_s3_url.
func (c *Client) DownloadCDN(ctx context.Context, rawURL string) (CDNResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("aimini: read cdn response: %w", err)
	}
	return CDNResponse(data), nil
}
