package aimini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strconv"
)

// QueueService calls the path-RPC queue API directly.
type QueueService struct {
	client *Client
}

// Add creates a queued generation task and returns the queued item.
func (s *QueueService) Add(ctx context.Context, req *AddQueueItemRequest) (*QueueAddResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("aimini: add request is required")
	}
	if req.Image == nil {
		return nil, fmt.Errorf("aimini: image is required")
	}
	if req.UserID == "" {
		return nil, fmt.Errorf("aimini: user id is required")
	}
	if req.NodeID == "" {
		return nil, fmt.Errorf("aimini: node id is required")
	}
	if len(req.Prompts) == 0 {
		return nil, fmt.Errorf("aimini: at least one prompt is required")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	image, filename, contentType, err := req.Image.open()
	if err != nil {
		return nil, fmt.Errorf("aimini: open image: %w", err)
	}
	defer image.Close()

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, escapeQuotes(filename)))
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, image); err != nil {
		return nil, fmt.Errorf("aimini: copy image: %w", err)
	}

	if err := writer.WriteField("user_id", req.UserID); err != nil {
		return nil, err
	}
	if err := writer.WriteField("node_id", req.NodeID); err != nil {
		return nil, err
	}
	prompts, err := json.Marshal(req.Prompts)
	if err != nil {
		return nil, fmt.Errorf("aimini: marshal prompts: %w", err)
	}
	if err := writer.WriteField("prompts", string(prompts)); err != nil {
		return nil, err
	}
	if req.NegativePrompt != "" {
		if err := writer.WriteField("negative_prompt", req.NegativePrompt); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.endpoint("/queue.add"), body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", s.client.config.Token)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, readAPIError(resp)
	}
	defer resp.Body.Close()

	var addResp QueueAddResponse
	if err := json.NewDecoder(resp.Body).Decode(&addResp); err != nil {
		return nil, fmt.Errorf("aimini: decode queue.add response: %w", err)
	}
	return &addResp, nil
}

// List returns queue items matching the request filter.
func (s *QueueService) List(ctx context.Context, req *ListQueueItemsRequest) (QueueListResponse, error) {
	if req == nil {
		req = &ListQueueItemsRequest{}
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("aimini: marshal queue.list request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.endpoint("/queue.list"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", s.client.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}
	defer resp.Body.Close()

	var items QueueListResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("aimini: decode queue.list response: %w", err)
	}
	return items, nil
}

// Delete deletes a queue item. 404 is treated as success because the item is
// already gone.
func (s *QueueService) Delete(ctx context.Context, req *DeleteQueueItemRequest) (*QueueDeleteResponse, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("aimini: queue item id is required")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("aimini: marshal queue.delete request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.endpoint("/queue.delete"), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", s.client.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return &QueueDeleteResponse{}, nil
	}
	return nil, readAPIError(resp)
}

// Take takes the next queued item. This is mainly for external workers or
// diagnostics.
func (s *QueueService) Take(ctx context.Context) (*QueueTakeResponse, error) {
	var out QueueTakeResponse
	if err := s.postJSON(ctx, "/queue.take", QueueTakeRequest{}, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MarkProcessing marks a queue item as processing.
func (s *QueueService) MarkProcessing(ctx context.Context, req *QueueProcessingRequest) (*QueueProcessingResponse, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("aimini: queue item id is required")
	}
	var out QueueProcessingResponse
	if err := s.postJSON(ctx, "/queue.processing", req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Complete marks a queue item processed using an existing output S3 key or URL.
func (s *QueueService) Complete(ctx context.Context, req *QueueCompleteJSONRequest) (*QueueCompleteResponse, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("aimini: queue item id is required")
	}
	var out QueueCompleteResponse
	if err := s.postJSON(ctx, "/queue.complete", req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CompleteWithImage marks a queue item processed by uploading an output image.
func (s *QueueService) CompleteWithImage(ctx context.Context, req *QueueCompleteMultipartRequest) (*QueueCompleteResponse, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("aimini: queue item id is required")
	}
	if req.Image == nil {
		return nil, fmt.Errorf("aimini: image is required")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	image, filename, contentType, err := req.Image.open()
	if err != nil {
		return nil, fmt.Errorf("aimini: open image: %w", err)
	}
	defer image.Close()

	partHeader := make(textproto.MIMEHeader)
	partHeader.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, escapeQuotes(filename)))
	partHeader.Set("Content-Type", contentType)
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(part, image); err != nil {
		return nil, fmt.Errorf("aimini: copy image: %w", err)
	}
	if err := writer.WriteField("id", req.ID); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.endpoint("/queue.complete"), body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", s.client.config.Token)
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, readAPIError(resp)
	}
	defer resp.Body.Close()

	var out QueueCompleteResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("aimini: decode queue.complete response: %w", err)
	}
	return &out, nil
}

// Fail marks a queue item failed.
func (s *QueueService) Fail(ctx context.Context, req *QueueFailRequest) (*QueueFailResponse, error) {
	if req == nil || req.ID == "" {
		return nil, fmt.Errorf("aimini: queue item id is required")
	}
	var out QueueFailResponse
	if err := s.postJSON(ctx, "/queue.fail", req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *QueueService) postJSON(ctx context.Context, endpoint string, in any, wantStatus int, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("aimini: marshal %s request: %w", endpoint, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.endpoint(endpoint), bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", s.client.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	if resp.StatusCode != wantStatus {
		return readAPIError(resp)
	}
	defer resp.Body.Close()

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("aimini: decode %s response: %w", endpoint, err)
	}
	return nil
}

func escapeQuotes(s string) string {
	q := strconv.Quote(s)
	return q[1 : len(q)-1]
}
