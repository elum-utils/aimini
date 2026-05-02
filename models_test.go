package aimini

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestGenerateContentQueuesAndPollsProcessedItem(t *testing.T) {
	t.Parallel()

	var listCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/queue.add":
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			if r.FormValue("user_id") != "user-default" {
				t.Fatalf("user_id = %q", r.FormValue("user_id"))
			}
			if r.FormValue("node_id") != "node-default" {
				t.Fatalf("node_id = %q", r.FormValue("node_id"))
			}
			if r.FormValue("prompts") != `["make it cinematic"]` {
				t.Fatalf("prompts = %q", r.FormValue("prompts"))
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"item-1","status":"queued"}`))
		case "/queue.list":
			var req ListQueueItemsRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if req.NodeID != "node-default" || req.Status != StatusProcessed {
				t.Fatalf("unexpected list request: %+v", req)
			}
			if listCalls.Add(1) == 1 {
				_, _ = w.Write([]byte(`[]`))
				return
			}
			_, _ = w.Write([]byte(`[{"id":"item-1","status":"processed","output_s3_url":"https://cdn/result.png"}]`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	client.config.PollInterval = time.Millisecond
	resp, err := client.Models.GenerateContent(
		context.Background(),
		"aimini-image",
		[]*Content{NewContent(Image(ImageFromBytes([]byte("img"), "input.png", "image/png")), Text("make it cinematic"))},
		nil,
	)
	if err != nil {
		t.Fatalf("GenerateContent returned error: %v", err)
	}
	if resp.OutputURL != "https://cdn/result.png" {
		t.Fatalf("OutputURL = %q", resp.OutputURL)
	}
	if resp.Item.ID != "item-1" {
		t.Fatalf("item id = %q", resp.Item.ID)
	}
}

func TestGenerateContentCanDeleteAfterResult(t *testing.T) {
	t.Parallel()

	var deleted atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/queue.add":
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"item-1","status":"queued"}`))
		case "/queue.list":
			_, _ = w.Write([]byte(`[{"id":"item-1","status":"processed","output_s3_url":"https://cdn/result.png"}]`))
		case "/queue.delete":
			deleted.Store(true)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	_, err := client.Models.GenerateContent(
		context.Background(),
		"aimini-image",
		[]*Content{NewContent(Image(ImageFromBytes([]byte("img"), "input.png", "image/png")), Text("prompt"))},
		&GenerateContentConfig{DeleteAfter: true},
	)
	if err != nil {
		t.Fatalf("GenerateContent returned error: %v", err)
	}
	if !deleted.Load() {
		t.Fatal("expected delete call")
	}
}

func TestCollectPartsValidatesInput(t *testing.T) {
	t.Parallel()

	if _, _, err := collectParts([]*Content{NewContent(Text("prompt"))}); err == nil {
		t.Fatal("expected missing image error")
	}
	if _, _, err := collectParts([]*Content{NewContent(Image(ImageFromBytes([]byte("a"), "a.png", "")))}); err == nil {
		t.Fatal("expected missing text error")
	}
	if _, _, err := collectParts([]*Content{NewContent(
		Image(ImageFromBytes([]byte("a"), "a.png", "")),
		Image(ImageFromBytes([]byte("b"), "b.png", "")),
		Text("prompt"),
	)}); err == nil {
		t.Fatal("expected duplicate image error")
	}
}
