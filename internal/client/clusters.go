package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// CreateCluster provisions a cluster of the given engine. It returns the new
// cluster ID and the action ID for the provisioning job, which can be awaited
// with WaitForAction.
func (c *Client) CreateCluster(ctx context.Context, in CreateClusterInput) (clusterID, actionID string, err error) {
	body, err := buildCreateBody(in)
	if err != nil {
		return "", "", err
	}
	var resp asyncResponse
	path := "/" + in.DBType.PathPrefix() + "/create"
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", "", err
	}
	if resp.ClusterID == "" {
		return "", "", fmt.Errorf("scalegrid: create cluster response did not include a clusterID")
	}
	return resp.ClusterID, resp.ActionID, nil
}

func buildCreateBody(in CreateClusterInput) (map[string]any, error) {
	if len(in.MachinePoolIDs) == 0 {
		return nil, fmt.Errorf("scalegrid: at least one cloud profile (machine pool) is required")
	}
	switch in.DBType {
	case DBMongo:
		body := map[string]any{
			"clusterName":       in.Name,
			"shardCount":        in.ShardCount,
			"replicaCount":      in.ReplicaCount,
			"size":              in.Size,
			"version":           strings.ToUpper(in.Version),
			"machinePoolIDList": in.MachinePoolIDs,
			"enableAuth":        true,
			"engine":            defaultStr(in.MongoEngine, "wiredtiger"),
			"enableSSL":         in.EnableSSL,
			"encryptDisk":       in.EncryptDisk,
		}
		if in.CompressionAlgo != "" {
			body["compressionAlgo"] = in.CompressionAlgo
		}
		return body, nil

	case DBRedis:
		redisConfig := map[string]any{}
		if in.EnableRDB {
			redisConfig["save"] = map[string]any{"value": "900 1 300 10 60 10000", "split": 0}
		} else {
			redisConfig["save"] = map[string]any{"value": "", "split": 0}
		}
		if in.EnableAOF {
			redisConfig["appendonly"] = map[string]any{"value": "yes", "split": 0}
		} else {
			redisConfig["appendonly"] = map[string]any{"value": "no", "split": 0}
		}
		redisConfig["maxmemory-policy"] = map[string]any{"value": defaultStr(in.MaxMemoryPolicy, "noeviction"), "split": 0}

		body := map[string]any{
			"clusterName":           in.Name,
			"version":               strings.ToUpper(in.Version),
			"size":                  in.Size,
			"serverCount":           in.ServerCount,
			"shardCount":            in.ShardCount,
			"machinePoolIDList":     in.MachinePoolIDs,
			"clusterMode":           in.ClusterMode,
			"backupIntervalInHours": in.BackupIntervalInHours,
			"encryptDisk":           in.EncryptDisk,
			"sentinelCount":         in.SentinelCount,
			"redisConfigParams":     redisConfig,
		}
		if len(in.CIDRList) > 0 {
			body["cidrList"] = in.CIDRList
		}
		if len(in.SentinelPools) > 0 {
			body["sentinelMachinePool"] = in.SentinelPools
		}
		return body, nil

	case DBMySQL:
		body := map[string]any{
			"clusterName":       in.Name,
			"shardCount":        in.ShardCount,
			"replicaCount":      in.ReplicaCount,
			"size":              in.Size,
			"version":           strings.ToLower(in.Version),
			"machinePoolIDList": in.MachinePoolIDs,
			"replicaConfig":     in.ReplicaConfig,
			"enableAuth":        true,
			"engine":            "INNODB",
			"enableSSL":         in.EnableSSL,
			"encryptDisk":       in.EncryptDisk,
		}
		if len(in.CIDRList) > 0 {
			body["cidrList"] = in.CIDRList
		}
		return body, nil

	case DBPostgreSQL:
		body := map[string]any{
			"clusterName":       in.Name,
			"shardCount":        in.ShardCount,
			"replicaCount":      in.ReplicaCount,
			"size":              in.Size,
			"version":           in.Version,
			"machinePoolIDList": in.MachinePoolIDs,
			"enableAuth":        true,
			"enableSSL":         in.EnableSSL,
			"encryptDisk":       in.EncryptDisk,
			"enablePgBouncer":   in.EnablePgBouncer,
		}
		if len(in.CIDRList) > 0 {
			body["cidrList"] = in.CIDRList
		}
		if in.ReplicaCount > 1 {
			body["replicationType"] = defaultStr(in.ReplicationType, "ASYNC")
			body["syncCommitType"] = defaultStr(in.SyncCommitType, "LOCAL")
		}
		return body, nil

	default:
		return nil, fmt.Errorf("scalegrid: unsupported database type %q", in.DBType)
	}
}

// GetCluster fetches a single cluster by ID.
func (c *Client) GetCluster(ctx context.Context, db DBType, id string) (*Cluster, error) {
	var resp clusterFetchResponse
	path := fmt.Sprintf("/%s/%s/fetch", db.PathPrefix(), id)
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Cluster.ID == "" {
		resp.Cluster.ID = id
	}
	return &resp.Cluster, nil
}

// ListClusters returns all clusters of the given engine.
func (c *Client) ListClusters(ctx context.Context, db DBType) ([]Cluster, error) {
	var resp clusterListResponse
	path := "/" + db.PathPrefix() + "/list"
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Clusters, nil
}

// FindClusterByName looks up a cluster of the given engine by name.
func (c *Client) FindClusterByName(ctx context.Context, db DBType, name string) (*Cluster, error) {
	clusters, err := c.ListClusters(ctx, db)
	if err != nil {
		return nil, err
	}
	for i := range clusters {
		if strings.EqualFold(clusters[i].Name, name) {
			return &clusters[i], nil
		}
	}
	return nil, &APIError{Code: "NotFound", Message: fmt.Sprintf("%s cluster %q was not found", db, name)}
}

// DeleteCluster tears down a cluster. When deleteVMs is true the underlying VMs
// are removed as well.
func (c *Client) DeleteCluster(ctx context.Context, db DBType, id string, deleteVMs bool) (string, error) {
	var resp asyncResponse
	path := fmt.Sprintf("/%s/%s", db.PathPrefix(), id)
	body := map[string]any{"skipVMDeletion": !deleteVMs}
	if err := c.do(ctx, http.MethodDelete, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// ScaleCluster changes the cluster size in place.
func (c *Client) ScaleCluster(ctx context.Context, db DBType, id, newSize string) (string, error) {
	var resp asyncResponse
	path := "/" + db.PathPrefix() + "/scale"
	body := map[string]any{"id": id, "newSize": newSize}
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// PauseCluster pauses a running cluster.
func (c *Client) PauseCluster(ctx context.Context, db DBType, id string) (string, error) {
	return c.clusterPowerAction(ctx, "/clusters/pauseCluster", db, id)
}

// ResumeCluster resumes a paused cluster.
func (c *Client) ResumeCluster(ctx context.Context, db DBType, id string) (string, error) {
	return c.clusterPowerAction(ctx, "/clusters/resumeCluster", db, id)
}

func (c *Client) clusterPowerAction(ctx context.Context, path string, db DBType, id string) (string, error) {
	var resp asyncResponse
	body := map[string]any{"clusterID": id, "dbType": db.WireType()}
	if err := c.do(ctx, http.MethodPost, path, body, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}

// GetCredentials returns the root credentials and connection strings.
func (c *Client) GetCredentials(ctx context.Context, db DBType, id string) (*Credentials, error) {
	var credResp credentialsResponse
	credPath := fmt.Sprintf("/%s/%s/getCredentials", db.PathPrefix(), id)
	if err := c.do(ctx, http.MethodGet, credPath, nil, &credResp); err != nil {
		return nil, err
	}

	creds := &Credentials{User: credResp.User, Password: credResp.Password}

	cluster, err := c.GetCluster(ctx, db, id)
	if err == nil {
		creds.ConnectionStrings = cluster.ConnectionString
		if db == DBPostgreSQL {
			creds.CommandLine = cluster.CommandLineServer
		} else {
			creds.CommandLine = cluster.CommandLineString
		}
	}
	return creds, nil
}

func defaultStr(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
