package aimini

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultPollInterval = 5 * time.Second
	defaultListLimit    = 50
)

// ClientConfig configures the API client. Token, BaseURL, NodeID, and UserID
// should be supplied by the application at runtime.
type ClientConfig struct {
	BaseURL string
	Token   string
	NodeID  string
	UserID  string

	HTTPClient   *http.Client
	PollInterval time.Duration
}

// Client is the root API client. Queue exposes the raw queue API. Models offers
// a higher-level image generation surface shaped similarly to model clients.
type Client struct {
	config ClientConfig

	httpClient *http.Client
	baseURL    *url.URL

	Queue  *QueueService
	Models *ModelsService
}

// NewClient creates a client and validates required configuration.
func NewClient(_ context.Context, config *ClientConfig) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("aimini: client config is required")
	}
	if strings.TrimSpace(config.BaseURL) == "" {
		return nil, fmt.Errorf("aimini: BaseURL is required")
	}
	if strings.TrimSpace(config.Token) == "" {
		return nil, fmt.Errorf("aimini: Token is required")
	}

	baseURL, err := url.Parse(strings.TrimRight(config.BaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("aimini: parse BaseURL: %w", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, fmt.Errorf("aimini: BaseURL must be absolute")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	pollInterval := config.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	cfg := *config
	cfg.BaseURL = baseURL.String()
	cfg.PollInterval = pollInterval

	client := &Client{
		config:     cfg,
		httpClient: httpClient,
		baseURL:    baseURL,
	}
	client.Queue = &QueueService{client: client}
	client.Models = &ModelsService{client: client}
	return client, nil
}

func (c *Client) endpoint(method string) string {
	ref := &url.URL{Path: method}
	return c.baseURL.ResolveReference(ref).String()
}
