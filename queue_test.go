package aimini

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestQueueAddSendsMultipartRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/queue.add" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %q", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "secret" {
			t.Fatalf("Authorization = %q", got)
		}

		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("MultipartReader: %v", err)
		}

		fields := map[string]string{}
		var imageBody string
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("NextPart: %v", err)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if part.FormName() == "image" {
				if part.FileName() != "input.jpg" {
					t.Fatalf("image filename = %q", part.FileName())
				}
				if got := part.Header.Get("Content-Type"); got != "image/jpeg" {
					t.Fatalf("image content type = %q", got)
				}
				imageBody = string(data)
				continue
			}
			fields[part.FormName()] = string(data)
		}

		if imageBody != "fake-jpeg" {
			t.Fatalf("image body = %q", imageBody)
		}
		if fields["user_id"] != "user-1" {
			t.Fatalf("user_id = %q", fields["user_id"])
		}
		if fields["node_id"] != "node-1" {
			t.Fatalf("node_id = %q", fields["node_id"])
		}
		if fields["prompts"] != `["prompt one","prompt two"]` {
			t.Fatalf("prompts = %q", fields["prompts"])
		}
		if fields["negative_prompt"] != "blur" {
			t.Fatalf("negative_prompt = %q", fields["negative_prompt"])
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"item-1","user_id":"user-1","node_id":"node-1","prompts":["prompt one","prompt two"],"status":"queued"}`))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	item, err := client.Queue.Add(context.Background(), &AddQueueItemRequest{
		Image:          ImageFromBytes([]byte("fake-jpeg"), "input.jpg", ""),
		UserID:         "user-1",
		NodeID:         "node-1",
		Prompts:        []string{"prompt one", "prompt two"},
		NegativePrompt: "blur",
	})
	if err != nil {
		t.Fatalf("Add returned error: %v", err)
	}
	if item.ID != "item-1" || item.Status != StatusQueued {
		t.Fatalf("unexpected item: %+v", item)
	}
}

func TestQueueListSendsJSONRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/queue.list" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q", got)
		}
		var req ListQueueItemsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if req.NodeID != "node-1" || req.Status != StatusProcessed || req.Limit != 10 {
			t.Fatalf("unexpected list request: %+v", req)
		}

		_, _ = w.Write([]byte(`[{"id":"item-1","status":"processed","output_s3_url":"https://cdn/result.png"}]`))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	items, err := client.Queue.List(context.Background(), &ListQueueItemsRequest{
		NodeID: "node-1",
		Status: StatusProcessed,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].OutputS3URL == "" {
		t.Fatalf("unexpected items: %+v", items)
	}
}

func TestQueueDeleteTreats404AsSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/queue.delete" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var req DeleteQueueItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Decode: %v", err)
		}
		if req.ID != "item-1" {
			t.Fatalf("id = %q", req.ID)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	if err := client.Queue.Delete(context.Background(), &DeleteQueueItemRequest{ID: "item-1"}); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
}

func TestQueueErrorIncludesMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad token"}`))
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)
	_, err := client.Queue.List(context.Background(), &ListQueueItemsRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "bad token") {
		t.Fatalf("error = %v", err)
	}
}

func TestAddRejectsInvalidRequest(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, "https://example.test")
	_, err := client.Queue.Add(context.Background(), &AddQueueItemRequest{})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

var _ = multipart.ErrMessageTooLarge

func newTestClient(t *testing.T, baseURL string) *Client {
	t.Helper()

	client, err := NewClient(context.Background(), &ClientConfig{
		BaseURL: baseURL,
		Token:   "secret",
		NodeID:  "node-default",
		UserID:  "user-default",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}
