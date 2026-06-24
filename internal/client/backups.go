package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ListBackups returns the backups for a cluster.
func (c *Client) ListBackups(ctx context.Context, db DBType, clusterID string) ([]Backup, error) {
	var resp backupListResponse
	// Redis serves this from a lowercase "listbackups" path; the others use
	// "listBackups".
	op := "listBackups"
	if db == DBRedis {
		op = "listbackups"
	}
	path := fmt.Sprintf("/%s/%s/%s", db.listPrefix(), clusterID, op)
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Backups, nil
}

// FindBackupByName looks up a backup of a cluster by name.
func (c *Client) FindBackupByName(ctx context.Context, db DBType, clusterID, name string) (*Backup, error) {
	backups, err := c.ListBackups(ctx, db, clusterID)
	if err != nil {
		return nil, err
	}
	for i := range backups {
		if strings.EqualFold(backups[i].Name, name) {
			return &backups[i], nil
		}
	}
	return nil, &APIError{Code: "NotFound", Message: fmt.Sprintf("backup %q was not found", name)}
}

// StartBackup triggers an on-demand backup. target may be "" or PRIMARY/SECONDARY
// (MASTER/SLAVE/STANDBY) for replica sets. For PostgreSQL the API requires both
// a type of "ONDEMAND" and a target, so a target defaulting to MASTER is sent.
func (c *Client) StartBackup(ctx context.Context, db DBType, clusterID, name, comment, target string) (string, error) {
	body := map[string]any{"backupName": name, "comment": comment, "id": clusterID}
	if db == DBPostgreSQL {
		body["type"] = "ONDEMAND"
		body["target"] = defaultStr(target, "MASTER")
	} else if target != "" {
		body["target"] = target
	}
	var resp asyncResponse
	path := "/" + db.listPrefix() + "/backup"
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// DeleteBackup removes a backup by ID.
func (c *Client) DeleteBackup(ctx context.Context, db DBType, clusterID, backupID string, force bool) (string, error) {
	body := map[string]any{"clusterID": clusterID, "backupID": backupID}
	// Only MongoDB and Redis accept a "force" flag.
	if db == DBMongo || db == DBRedis {
		body["force"] = force
	}
	var resp asyncResponse
	// PostgreSQL uses the capitalized prefix with an all-lowercase "deletebackup"
	// segment; the others use "<Prefix>/deleteBackup".
	path := "/" + db.PathPrefix() + "/deleteBackup"
	if db == DBPostgreSQL {
		path = "/" + db.PathPrefix() + "/deletebackup"
	}
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// RestoreBackup restores a backup onto its cluster.
func (c *Client) RestoreBackup(ctx context.Context, db DBType, clusterID, backupID string) (string, error) {
	body := map[string]any{"clusterID": clusterID, "backupID": backupID}
	var resp asyncResponse
	path := "/" + db.listPrefix() + "/restore"
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// SetBackupSchedule configures (or, when enabled is false, disables) scheduled
// backups for a cluster.
func (c *Client) SetBackupSchedule(ctx context.Context, db DBType, clusterID string, enabled bool, intervalHours, hour, limit int, target string) error {
	// MongoDB and Redis name this endpoint "setClusterBackupSchedule"; MySQL and
	// PostgreSQL use "setBackupSchedule".
	op := "setBackupSchedule"
	if db == DBMongo || db == DBRedis {
		op = "setClusterBackupSchedule"
	}
	path := "/" + db.listPrefix() + "/" + op
	body := map[string]any{"id": clusterID}
	if enabled {
		body["scheduledBackupEnabled"] = true
		body["backupIntervalInHours"] = intervalHours
		body["backupHour"] = hour
		body["backupScheduledBackupLimit"] = limit
		if target != "" {
			body["target"] = target
		}
	}
	return c.do(ctx, http.MethodPost, path, body, nil)
}
