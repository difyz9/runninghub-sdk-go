package runninghub

import (
	"context"
	"strings"
	"time"
)

const defaultPollInterval = 2 * time.Second

// WaitForTask polls QueryTaskV2 until the task reaches a terminal state or ctx is done.
//
// Terminal states are SUCCESS, FAILED, and CANCELLED.
func (c *Client) WaitForTask(ctx context.Context, taskID string, pollInterval time.Duration) (*QueryV2Response, error) {
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	return c.waitForTaskCompletion(ctx, taskID, pollInterval, nil)
}

func isTerminalTaskStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESS", "FAILED", "CANCELLED":
		return true
	default:
		return false
	}
}