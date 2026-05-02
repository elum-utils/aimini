package aimini

import "time"

const (
	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusProcessed  = "processed"
	StatusFailed     = "failed"
)

// QueueItem is the API representation of an image generation queue item.
type QueueItem struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	NodeID         string     `json:"node_id"`
	Prompts        []string   `json:"prompts"`
	NegativePrompt string     `json:"negative_prompt"`
	InputS3Key     string     `json:"input_s3_key"`
	InputS3URL     string     `json:"input_s3_url"`
	OutputS3Key    string     `json:"output_s3_key"`
	OutputS3URL    string     `json:"output_s3_url"`
	Status         string     `json:"status"`
	CreatedAt      *time.Time `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
	ProcessedAt    *time.Time `json:"processed_at"`
}

// AddQueueItemRequest creates a generation job with an input image.
type AddQueueItemRequest struct {
	Image          ImageSource
	UserID         string
	NodeID         string
	Prompts        []string
	NegativePrompt string
}

// ListQueueItemsRequest filters queue items.
type ListQueueItemsRequest struct {
	NodeID string `json:"node_id,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// DeleteQueueItemRequest deletes an item after successful delivery.
type DeleteQueueItemRequest struct {
	ID string `json:"id"`
}
