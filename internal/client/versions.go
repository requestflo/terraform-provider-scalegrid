package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// GetDatabaseVersions returns the available database versions for an engine and
// cloud provider. cloudProvider should be one of AWS, AZURE, DO, or GCP.
func (c *Client) GetDatabaseVersions(ctx context.Context, db DBType, cloudProvider string) ([]string, error) {
	cloud, err := normalizeCloudProvider(cloudProvider)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("dbType", db.WireType())
	q.Set("cloudProvider", cloud)

	var resp databaseVersionsResponse
	path := "/Clusters/getDatabaseActiveVersions?" + q.Encode()
	if err := c.do(ctx, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	return resp.Versions, nil
}

func normalizeCloudProvider(s string) (string, error) {
	switch s {
	case "AWS", "aws":
		return "EC2", nil
	case "AZURE", "azure":
		return "AZUREARM", nil
	case "DO", "do", "digitalocean", "DIGITALOCEAN":
		return "DIGITALOCEAN", nil
	case "GCP", "gcp", "google":
		return "GCP", nil
	default:
		return "", fmt.Errorf("scalegrid: unsupported cloud provider %q (use AWS, AZURE, DO, or GCP)", s)
	}
}
