package client

import (
	"context"
	"net/http"
)

// GetFirewallRules returns the cluster-level IP whitelist (CIDR list).
func (c *Client) GetFirewallRules(ctx context.Context, db DBType, clusterID string) ([]string, error) {
	var resp firewallResponse
	body := map[string]any{"clusterID": clusterID, "dbType": db.WireType()}
	if err := c.do(ctx, http.MethodPost, "/Clusters/getClusterLevelIPWhiteList", body, &resp); err != nil {
		return nil, err
	}
	return resp.CIDRList, nil
}

// SetFirewallRules replaces the cluster-level IP whitelist. The console requires
// two calls: one to record the cluster-level list and one to apply it.
func (c *Client) SetFirewallRules(ctx context.Context, db DBType, clusterID string, cidrs []string) error {
	if cidrs == nil {
		cidrs = []string{}
	}
	body := map[string]any{"clusterID": clusterID, "dbType": db.WireType(), "cidrList": cidrs}
	if err := c.do(ctx, http.MethodPost, "/Clusters/setClusterLevelIPWhiteList", body, nil); err != nil {
		return err
	}
	return c.do(ctx, http.MethodPost, "/Clusters/configureIPWhiteList", body, nil)
}
