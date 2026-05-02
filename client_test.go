package aimini

import (
	"context"
	"testing"
)

func TestNewClientValidatesConfig(t *testing.T) {
	t.Parallel()

	if _, err := NewClient(context.Background(), nil); err == nil {
		t.Fatal("expected nil config error")
	}
	if _, err := NewClient(context.Background(), &ClientConfig{Token: "token"}); err == nil {
		t.Fatal("expected base url error")
	}
	if _, err := NewClient(context.Background(), &ClientConfig{BaseURL: "https://example.test"}); err == nil {
		t.Fatal("expected token error")
	}
	if _, err := NewClient(context.Background(), &ClientConfig{BaseURL: "://bad", Token: "token"}); err == nil {
		t.Fatal("expected invalid base url error")
	}
}

func TestNewClientDefaults(t *testing.T) {
	t.Parallel()

	client, err := NewClient(context.Background(), &ClientConfig{
		BaseURL: "https://example.test/",
		Token:   "token",
	})
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}
	if client.Queue == nil {
		t.Fatal("Queue service is nil")
	}
	if client.Models == nil {
		t.Fatal("Models service is nil")
	}
	if got := client.endpoint("/queue.list"); got != "https://example.test/queue.list" {
		t.Fatalf("endpoint = %q", got)
	}
	if client.config.PollInterval != defaultPollInterval {
		t.Fatalf("poll interval = %v", client.config.PollInterval)
	}
}
