package client

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// FollowerStatus describes a follower relationship.
type FollowerStatus struct {
	DestinationCluster Cluster `json:"destinationCluster"`
	SourceCluster      Cluster `json:"sourceCluster"`
	SyncSchedule       struct {
		IntervalInHours int    `json:"intervalInHours"`
		JobType         string `json:"jobType"`
		NextRuntime     int64  `json:"nextRuntime"`
	} `json:"syncSchedule"`
}

// CreateFollower establishes a follower relationship: targetClusterID follows
// sourceClusterID, syncing every intervalHours starting at startHour (0-23).
func (c *Client) CreateFollower(ctx context.Context, db DBType, targetClusterID, sourceClusterID string, intervalHours, startHour int) error {
	start := time.Now().Truncate(time.Hour).Add(time.Duration(startHour) * time.Hour)
	body := map[string]any{
		"sourceClusterID": sourceClusterID,
		"dbType":          db.WireType(),
		"intervalInHours": fmt.Sprintf("%d", intervalHours),
		"startTimeStr":    start.Format("2006-01-02T15:04:05"),
	}
	path := fmt.Sprintf("/clusters/%s/createFollowerRelationship", targetClusterID)
	return c.do(ctx, http.MethodPost, path, body, nil)
}

// GetFollowerStatus returns the follower relationship info for a target cluster.
func (c *Client) GetFollowerStatus(ctx context.Context, db DBType, targetClusterID string) (*FollowerStatus, error) {
	body := map[string]any{"dbType": db.WireType()}
	var resp FollowerStatus
	path := fmt.Sprintf("/clusters/%s/getFollowerRelationshipInfo", targetClusterID)
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SyncFollowerNow triggers an on-demand sync of a follower cluster.
func (c *Client) SyncFollowerNow(ctx context.Context, db DBType, targetClusterID string) (string, error) {
	body := map[string]any{"dbType": db.WireType()}
	var resp asyncResponse
	path := fmt.Sprintf("/clusters/%s/syncFollowerClusterNow", targetClusterID)
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// BreakFollower breaks the follower relationship of a target cluster.
func (c *Client) BreakFollower(ctx context.Context, db DBType, targetClusterID string) error {
	body := map[string]any{"dbType": db.WireType()}
	path := fmt.Sprintf("/clusters/%s/breakFollowerRelationship", targetClusterID)
	return c.do(ctx, http.MethodPost, path, body, nil)
}
