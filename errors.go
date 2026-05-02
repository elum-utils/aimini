package aimini

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxErrorBodyBytes = 1 << 20

// Error is returned for non-successful API responses.
type Error struct {
	StatusCode int
	Status     string
	Message    string
	Body       string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("aimini: %s: %s", e.Status, e.Message)
	}
	if e.Body != "" {
		return fmt.Sprintf("aimini: %s: %s", e.Status, e.Body)
	}
	return fmt.Sprintf("aimini: %s", e.Status)
}

func readAPIError(resp *http.Response) error {
	defer resp.Body.Close()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if readErr != nil {
		return fmt.Errorf("aimini: read error response: %w", readErr)
	}

	apiErr := &Error{
		StatusCode: resp.StatusCode,
		Status:     resp.Status,
		Body:       strings.TrimSpace(string(body)),
	}

	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if len(body) > 0 && json.Unmarshal(body, &payload) == nil {
		if payload.Message != "" {
			apiErr.Message = payload.Message
		} else if payload.Error != "" {
			apiErr.Message = payload.Error
		}
	}

	return apiErr
}
