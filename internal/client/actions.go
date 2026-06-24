package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// GetAction returns the status of an asynchronous job.
func (c *Client) GetAction(ctx context.Context, actionID string) (*Action, error) {
	var resp actionResponse
	if err := c.do(ctx, http.MethodGet, "/actions/"+actionID, nil, &resp); err != nil {
		return nil, err
	}
	return &resp.Action, nil
}

// WaitForAction polls an action until it completes or fails. A blank actionID is
// treated as immediately complete (some endpoints act synchronously).
func (c *Client) WaitForAction(ctx context.Context, actionID string, pollInterval time.Duration) error {
	if actionID == "" {
		return nil
	}
	if pollInterval <= 0 {
		pollInterval = 20 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		action, err := c.GetAction(ctx, actionID)
		if err != nil {
			return err
		}
		// The API reports status as Running/Completed/Failed; compare
		// case-insensitively to be resilient to casing changes.
		switch {
		case strings.EqualFold(action.Status, ActionCompleted):
			return nil
		case strings.EqualFold(action.Status, ActionFailed) || action.Cancelled:
			msg := action.StepError.ErrorMessageWithDetails
			if msg == "" {
				msg = "job failed"
			}
			return fmt.Errorf("scalegrid: action %s failed: %s", actionID, msg)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("scalegrid: timed out waiting for action %s: %w", actionID, ctx.Err())
		case <-ticker.C:
		}
	}
}
