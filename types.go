package aimini

const (
	QueueStatusQueued     = "queued"
	QueueStatusProcessing = "processing"
	QueueStatusProcessed  = "processed"
	QueueStatusFailed     = "failed"

	StatusQueued     = "queued"
	StatusProcessing = "processing"
	StatusProcessed  = "processed"
	StatusFailed     = "failed"
)

// HealthResponse is returned by GET /health.
type HealthResponse struct {
	OK bool `json:"ok"`
}

// ErrorResponse is the default API error response shape.
type ErrorResponse struct {
	Error string `json:"error"`
}

// QueueItem is the API representation of an image generation queue item.
type QueueItem struct {
	ID             string   `json:"id"`
	UserID         string   `json:"user_id"`
	NodeID         string   `json:"node_id"`
	Prompts        []string `json:"prompts"`
	NegativePrompt string   `json:"negative_prompt"`
	InputS3Key     string   `json:"input_s3_key"`
	InputS3URL     string   `json:"input_s3_url"`
	OutputS3Key    string   `json:"output_s3_key,omitempty"`
	OutputS3URL    string   `json:"output_s3_url,omitempty"`
	Status         string   `json:"status"`
	Error          string   `json:"error,omitempty"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
	ProcessedAt    string   `json:"processed_at,omitempty"`
}

// AddQueueItemRequest creates a generation job with an input image.
type AddQueueItemRequest struct {
	Image          ImageSource
	UserID         string
	NodeID         string
	Prompts        []string
	NegativePrompt string
}

// QueueAddResponse is returned by POST /queue.add.
type QueueAddResponse struct {
	Item      QueueItem `json:"item"`
	QueueSize int64     `json:"queue_size"`
}

// ListQueueItemsRequest filters queue items.
type ListQueueItemsRequest struct {
	NodeID string `json:"node_id,omitempty"`
	Status string `json:"status,omitempty"`
	Limit  int    `json:"limit,omitempty"`
}

// QueueListResponse is returned by POST /queue.list.
type QueueListResponse []QueueItem

// QueueTakeRequest is sent to POST /queue.take.
type QueueTakeRequest struct{}

// QueueTakeResponse is returned by POST /queue.take.
type QueueTakeResponse struct {
	Item *QueueItem `json:"item"`
}

// QueueProcessingRequest is sent to POST /queue.processing.
type QueueProcessingRequest struct {
	ID string `json:"id" form:"id"`
}

// QueueProcessingResponse is returned by POST /queue.processing.
type QueueProcessingResponse = QueueItem

// QueueCompleteJSONRequest is sent to POST /queue.complete as JSON.
type QueueCompleteJSONRequest struct {
	ID             string `json:"id"`
	ProcessedS3Key string `json:"processed_s3_key"`
	ProcessedS3URL string `json:"processed_s3_url"`
}

// QueueCompleteMultipartRequest is sent to POST /queue.complete as multipart.
type QueueCompleteMultipartRequest struct {
	ID    string
	Image ImageSource
}

// QueueCompleteResponse is returned by POST /queue.complete.
type QueueCompleteResponse = QueueItem

// DeleteQueueItemRequest deletes an item after successful delivery.
type DeleteQueueItemRequest struct {
	ID string `json:"id" form:"id"`
}

// QueueDeleteResponse is returned by POST /queue.delete. Successful responses
// have HTTP 204 and an empty body.
type QueueDeleteResponse struct{}

// QueueFailRequest is sent to POST /queue.fail.
type QueueFailRequest struct {
	ID    string `json:"id" form:"id"`
	Error string `json:"error" form:"error"`
}

// QueueFailResponse is returned by POST /queue.fail.
type QueueFailResponse = QueueItem

// CDNResponse is returned by GET /cdn/* as raw image bytes.
type CDNResponse []byte
