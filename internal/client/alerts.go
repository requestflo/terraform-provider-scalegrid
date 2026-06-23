package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// CreateAlertRuleInput holds the fields for a new alert rule.
type CreateAlertRuleInput struct {
	DBType        DBType
	ClusterID     string
	Type          string // METRIC, DISK_FREE, ROLE_CHANGE
	Metric        string // required when Type == METRIC
	Operator      string // EQ, LT, GT, LTE, GTE
	Threshold     string
	Notifications []string // EMAIL, SMS, PAGERDUTY
	Duration      string   // TWO, SIX, HOURLY, DAILY
}

// CreateAlertRule creates an alert rule and returns it.
func (c *Client) CreateAlertRule(ctx context.Context, in CreateAlertRuleInput) (*AlertRule, error) {
	body := map[string]any{
		"clusterId":     in.ClusterID,
		"databaseType":  in.DBType.WireType(),
		"alertRuleType": strings.ToUpper(in.Type),
		"operator":      strings.ToUpper(in.Operator),
		"threshold":     in.Threshold,
		"notifications": upperAll(in.Notifications),
	}
	if in.Metric != "" {
		body["metric"] = strings.ToUpper(in.Metric)
	}
	if in.Duration != "" {
		body["averageType"] = strings.ToUpper(in.Duration)
	}

	var resp alertRuleCreateResponse
	if err := c.do(ctx, http.MethodPost, "/AlertRules/create", body, &resp); err != nil {
		return nil, err
	}
	if resp.Rule.ID == "" {
		return nil, fmt.Errorf("scalegrid: create alert rule response did not include an id")
	}
	return &resp.Rule, nil
}

// ListAlertRules returns the alert rules for a cluster.
func (c *Client) ListAlertRules(ctx context.Context, db DBType, clusterID string) ([]AlertRule, error) {
	body := map[string]any{"clusterId": clusterID, "databaseType": db.WireType()}
	var resp struct {
		Rules []AlertRule `json:"rules"`
	}
	if err := c.do(ctx, http.MethodPost, "/AlertRules/list", body, &resp); err != nil {
		return nil, err
	}
	return resp.Rules, nil
}

// GetAlertRule fetches a single alert rule by ID for a cluster.
func (c *Client) GetAlertRule(ctx context.Context, db DBType, clusterID, ruleID string) (*AlertRule, error) {
	rules, err := c.ListAlertRules(ctx, db, clusterID)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ID == ruleID {
			return &rules[i], nil
		}
	}
	return nil, &APIError{Code: "NotFound", Message: fmt.Sprintf("alert rule %q was not found", ruleID)}
}

// DeleteAlertRule removes an alert rule.
func (c *Client) DeleteAlertRule(ctx context.Context, ruleID string, force bool) error {
	body := map[string]any{"forceDelete": force}
	return c.do(ctx, http.MethodDelete, "/AlertRules/"+ruleID, body, nil)
}

// UnmarshalJSON normalizes the notifications field, which the API returns as a
// stringified list.
func (r *AlertRule) UnmarshalJSON(data []byte) error {
	type alias AlertRule
	aux := struct {
		Notifications json.RawMessage `json:"notifications"`
		*alias
	}{alias: (*alias)(r)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Notifications) > 0 {
		var list []string
		if err := json.Unmarshal(aux.Notifications, &list); err == nil {
			r.Notifications = list
		} else {
			var s string
			if err := json.Unmarshal(aux.Notifications, &s); err == nil {
				s = strings.Trim(s, "[]")
				for _, part := range strings.Split(s, ",") {
					part = strings.TrimSpace(strings.Trim(part, "'\""))
					if part != "" {
						r.Notifications = append(r.Notifications, part)
					}
				}
			}
		}
	}
	return nil
}

func upperAll(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strings.ToUpper(v)
	}
	return out
}
