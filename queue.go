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
func (s *QueueService) Add(ctx context.Context, req *AddQueueItemRequest) (*QueueItem, error) {
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

	var item QueueItem
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, fmt.Errorf("aimini: decode queue.add response: %w", err)
	}
	return &item, nil
}

// List returns queue items matching the request filter.
func (s *QueueService) List(ctx context.Context, req *ListQueueItemsRequest) ([]*QueueItem, error) {
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

	var items []*QueueItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("aimini: decode queue.list response: %w", err)
	}
	return items, nil
}

// Delete deletes a queue item. 404 is treated as success because the item is
// already gone.
func (s *QueueService) Delete(ctx context.Context, req *DeleteQueueItemRequest) error {
	if req == nil || req.ID == "" {
		return fmt.Errorf("aimini: queue item id is required")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("aimini: marshal queue.delete request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.client.endpoint("/queue.delete"), bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", s.client.config.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusNotFound {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		return nil
	}
	return readAPIError(resp)
}

func escapeQuotes(s string) string {
	q := strconv.Quote(s)
	return q[1 : len(q)-1]
}
