package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// ListCloudProfiles returns all cloud profiles on the account.
func (c *Client) ListCloudProfiles(ctx context.Context) ([]CloudProfile, error) {
	var resp cloudProfileListResponse
	if err := c.do(ctx, http.MethodGet, "/clouds/list", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Clouds, nil
}

// FindCloudProfileByName looks up a cloud profile by its (case-insensitive) name.
func (c *Client) FindCloudProfileByName(ctx context.Context, name string) (*CloudProfile, error) {
	profiles, err := c.ListCloudProfiles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		if strings.EqualFold(profiles[i].Name, name) {
			return &profiles[i], nil
		}
	}
	return nil, &APIError{Code: "NotFound", Message: fmt.Sprintf("cloud profile %q was not found", name)}
}

// GetCloudProfile fetches a cloud profile by ID.
func (c *Client) GetCloudProfile(ctx context.Context, id string) (*CloudProfile, error) {
	profiles, err := c.ListCloudProfiles(ctx)
	if err != nil {
		return nil, err
	}
	for i := range profiles {
		if profiles[i].ID == id {
			return &profiles[i], nil
		}
	}
	return nil, &APIError{Code: "NotFound", Message: fmt.Sprintf("cloud profile %q was not found", id)}
}

// CreateAWSCloudProfile registers an AWS (EC2/VPC) cloud profile.
func (c *Client) CreateAWSCloudProfile(ctx context.Context, in CreateAWSCloudProfileInput) (machinePoolID, actionID string, err error) {
	body := map[string]any{
		"accessKey":          in.AccessKey,
		"secretKey":          in.SecretKey,
		"database":           in.DBType.WireType(),
		"dbType":             in.DBType.WireType(),
		"region":             strings.ToLower(in.Region),
		"deploymentStyle":    "VPC",
		"connectivityConfig": defaultStr(in.ConnectivityConfig, "INTERNET"),
		"name":               in.Name,
		"vpcID":              in.VPCID,
		"vpcSubnetID":        in.SubnetID,
		"vpcCIDR":            in.VPCCIDR,
		"vpcSubnetCIDR":      in.SubnetCIDR,
		"vpcSecurityGroupID": in.SecurityGroupID,
		"vpcSecurityGroup":   in.SecurityGroupName,
		"enableSSH":          in.EnableSSH,
	}
	var resp asyncResponse
	if err := c.do(ctx, http.MethodPost, "/clouds/createMachinePoolForEC2", body, &resp); err != nil {
		return "", "", err
	}
	if resp.MachinePoolID == "" {
		return "", "", fmt.Errorf("scalegrid: create cloud profile response did not include a machinePoolID")
	}
	return resp.MachinePoolID, resp.ActionID, nil
}

// UpdateAWSCloudProfileKeys rotates the access/secret keys on an AWS profile.
func (c *Client) UpdateAWSCloudProfileKeys(ctx context.Context, machinePoolID, accessKey, secretKey string) error {
	body := map[string]any{"machinePoolID": machinePoolID, "accessKey": accessKey, "secretKey": secretKey}
	return c.do(ctx, http.MethodPost, "/Clouds/updateEC2MachinePoolKeys", body, nil)
}

// DeleteCloudProfile removes a cloud profile.
func (c *Client) DeleteCloudProfile(ctx context.Context, id string) (string, error) {
	var resp asyncResponse
	if err := c.do(ctx, http.MethodDelete, "/clouds/"+id, nil, &resp); err != nil {
		return "", err
	}
	return resp.ActionID, nil
}
