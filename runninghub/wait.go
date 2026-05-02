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
	if ctx == nil {
		ctx = context.Background()
	}
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	for {
		out, err := c.QueryTaskV2(ctx, taskID)
		if err != nil {
			return nil, err
		}
		if isTerminalTaskStatus(out.Status) {
			return out, nil
		}

		timer := time.NewTimer(pollInterval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func isTerminalTaskStatus(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "SUCCESS", "FAILED", "CANCELLED":
		return true
	default:
		return false
	}
}