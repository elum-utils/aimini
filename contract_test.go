package aimini

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContractResponses(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			_, _ = w.Write([]byte(`{"ok":true}`))
		case "/queue.take":
			_, _ = w.Write([]byte(`{"item":{"id":"take-1","status":"queued","created_at":"2026-05-03T00:00:00Z","updated_at":"2026-05-03T00:00:00Z"}}`))
		case "/queue.processing":
			assertJSONID(t, r, "processing-1")
			_, _ = w.Write([]byte(`{"id":"processing-1","status":"processing","created_at":"2026-05-03T00:00:00Z","updated_at":"2026-05-03T00:00:01Z"}`))
		case "/queue.complete":
			assertJSONID(t, r, "complete-1")
			_, _ = w.Write([]byte(`{"id":"complete-1","status":"processed","output_s3_key":"output/result.png","output_s3_url":"https://cdn/result.png","created_at":"2026-05-03T00:00:00Z","updated_at":"2026-05-03T00:00:01Z","processed_at":"2026-05-03T00:00:01Z"}`))
		case "/queue.fail":
			assertJSONID(t, r, "fail-1")
			_, _ = w.Write([]byte(`{"id":"fail-1","status":"failed","error":"boom","created_at":"2026-05-03T00:00:00Z","updated_at":"2026-05-03T00:00:01Z"}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)

	client := newTestClient(t, server.URL)

	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !health.OK {
		t.Fatal("expected healthy response")
	}

	takeResp, err := client.Queue.Take(context.Background())
	if err != nil {
		t.Fatalf("Take returned error: %v", err)
	}
	if takeResp.Item == nil || takeResp.Item.ID != "take-1" {
		t.Fatalf("unexpected take response: %+v", takeResp)
	}

	processingResp, err := client.Queue.MarkProcessing(context.Background(), &QueueProcessingRequest{ID: "processing-1"})
	if err != nil {
		t.Fatalf("MarkProcessing returned error: %v", err)
	}
	if processingResp.Status != StatusProcessing {
		t.Fatalf("processing status = %q", processingResp.Status)
	}

	completeResp, err := client.Queue.Complete(context.Background(), &QueueCompleteJSONRequest{
		ID:             "complete-1",
		ProcessedS3Key: "output/result.png",
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if completeResp.OutputS3URL != "https://cdn/result.png" {
		t.Fatalf("complete output url = %q", completeResp.OutputS3URL)
	}

	failResp, err := client.Queue.Fail(context.Background(), &QueueFailRequest{
		ID:    "fail-1",
		Error: "boom",
	})
	if err != nil {
		t.Fatalf("Fail returned error: %v", err)
	}
	if failResp.Error != "boom" {
		t.Fatalf("fail error = %q", failResp.Error)
	}
}

func assertJSONID(t *testing.T, r *http.Request, want string) {
	t.Helper()

	var req struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if req.ID != want {
		t.Fatalf("id = %q, want %q", req.ID, want)
	}
}
