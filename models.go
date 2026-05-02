package aimini

import (
	"context"
	"fmt"
	"time"
)

// ModelsService provides a convenience surface for code that already thinks in
// terms of model generation instead of queue operations.
type ModelsService struct {
	client *Client
}

// GenerateContentConfig configures high-level image generation.
type GenerateContentConfig struct {
	UserID         string
	NodeID         string
	NegativePrompt string
	PollInterval   time.Duration
	ListLimit      int
	DeleteAfter    bool
}

// Part is a prompt or image part, similar to common multimodal model clients.
type Part struct {
	Text  string
	Image ImageSource
}

// Content groups parts into a request.
type Content struct {
	Parts []*Part
}

// Candidate is one generated result.
type Candidate struct {
	Content *Content
	Item    *QueueItem
}

// GenerateContentResponse contains the processed queue item and output URL.
type GenerateContentResponse struct {
	Candidates []*Candidate
	Item       *QueueItem
	OutputURL  string
}

// Text creates a text prompt part.
func Text(text string) *Part {
	return &Part{Text: text}
}

// Image creates an image part.
func Image(image ImageSource) *Part {
	return &Part{Image: image}
}

// NewContent creates a content group.
func NewContent(parts ...*Part) *Content {
	return &Content{Parts: parts}
}

// GenerateContent queues an image generation request and waits until that item
// appears in the processed list. The model argument is accepted for API
// compatibility and future routing; the current server routes by queue only.
func (s *ModelsService) GenerateContent(ctx context.Context, _ string, contents []*Content, config *GenerateContentConfig) (*GenerateContentResponse, error) {
	if config == nil {
		config = &GenerateContentConfig{}
	}

	userID := config.UserID
	if userID == "" {
		userID = s.client.config.UserID
	}
	nodeID := config.NodeID
	if nodeID == "" {
		nodeID = s.client.config.NodeID
	}

	image, prompts, err := collectParts(contents)
	if err != nil {
		return nil, err
	}

	item, err := s.client.Queue.Add(ctx, &AddQueueItemRequest{
		Image:          image,
		UserID:         userID,
		NodeID:         nodeID,
		Prompts:        prompts,
		NegativePrompt: config.NegativePrompt,
	})
	if err != nil {
		return nil, err
	}

	processed, err := s.WaitForResult(ctx, item.ID, &WaitForResultOptions{
		NodeID:       nodeID,
		PollInterval: config.PollInterval,
		ListLimit:    config.ListLimit,
	})
	if err != nil {
		return nil, err
	}

	if config.DeleteAfter {
		if err := s.client.Queue.Delete(ctx, &DeleteQueueItemRequest{ID: processed.ID}); err != nil {
			return nil, err
		}
	}

	return &GenerateContentResponse{
		Candidates: []*Candidate{{
			Content: NewContent(Text(processed.OutputS3URL)),
			Item:    processed,
		}},
		Item:      processed,
		OutputURL: processed.OutputS3URL,
	}, nil
}

// WaitForResultOptions configures result polling.
type WaitForResultOptions struct {
	NodeID       string
	PollInterval time.Duration
	ListLimit    int
}

// WaitForResult polls processed items until the given queue item id appears.
func (s *ModelsService) WaitForResult(ctx context.Context, id string, options *WaitForResultOptions) (*QueueItem, error) {
	if id == "" {
		return nil, fmt.Errorf("aimini: queue item id is required")
	}
	if options == nil {
		options = &WaitForResultOptions{}
	}
	nodeID := options.NodeID
	if nodeID == "" {
		nodeID = s.client.config.NodeID
	}
	limit := options.ListLimit
	if limit <= 0 {
		limit = defaultListLimit
	}
	pollInterval := options.PollInterval
	if pollInterval <= 0 {
		pollInterval = s.client.config.PollInterval
	}

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}

		items, err := s.client.Queue.List(ctx, &ListQueueItemsRequest{
			NodeID: nodeID,
			Status: StatusProcessed,
			Limit:  limit,
		})
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item != nil && item.ID == id {
				if item.OutputS3URL == "" {
					return nil, fmt.Errorf("aimini: processed item %q has empty output url", id)
				}
				return item, nil
			}
		}

		timer.Reset(pollInterval)
	}
}

func collectParts(contents []*Content) (ImageSource, []string, error) {
	var image ImageSource
	var prompts []string
	for _, content := range contents {
		if content == nil {
			continue
		}
		for _, part := range content.Parts {
			if part == nil {
				continue
			}
			if part.Image != nil {
				if image != nil {
					return nil, nil, fmt.Errorf("aimini: only one image part is supported")
				}
				image = part.Image
			}
			if part.Text != "" {
				prompts = append(prompts, part.Text)
			}
		}
	}
	if image == nil {
		return nil, nil, fmt.Errorf("aimini: image part is required")
	}
	if len(prompts) == 0 {
		return nil, nil, fmt.Errorf("aimini: text prompt is required")
	}
	return image, prompts, nil
}
