package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource                = (*alertRuleResource)(nil)
	_ resource.ResourceWithConfigure   = (*alertRuleResource)(nil)
	_ resource.ResourceWithImportState = (*alertRuleResource)(nil)
)

// NewAlertRuleResource is the constructor registered with the provider.
func NewAlertRuleResource() resource.Resource { return &alertRuleResource{} }

type alertRuleResource struct {
	client *client.Client
}

type alertRuleResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Database      types.String `tfsdk:"database"`
	ClusterID     types.String `tfsdk:"cluster_id"`
	Type          types.String `tfsdk:"type"`
	Metric        types.String `tfsdk:"metric"`
	Operator      types.String `tfsdk:"operator"`
	Threshold     types.String `tfsdk:"threshold"`
	Duration      types.String `tfsdk:"duration"`
	Notifications types.List   `tfsdk:"notifications"`
}

func (r *alertRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *alertRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an alert rule on a ScaleGrid cluster. Alert rules are immutable; changing " +
			"any attribute replaces the rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Unique identifier of the alert rule.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database": schema.StringAttribute{
				Required:      true,
				Description:   "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"cluster_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the cluster the rule applies to.",
				PlanModifiers: reqReplaceStr(),
			},
			"type": schema.StringAttribute{
				Required:      true,
				Description:   "Alert rule type: `METRIC`, `DISK_FREE`, or `ROLE_CHANGE`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("METRIC", "DISK_FREE", "ROLE_CHANGE")},
			},
			"metric": schema.StringAttribute{
				Optional:      true,
				Description:   "Metric name (required when type is METRIC). See ScaleGrid docs for valid metrics per engine.",
				PlanModifiers: reqReplaceStr(),
			},
			"operator": schema.StringAttribute{
				Required:      true,
				Description:   "Comparison operator: `EQ`, `LT`, `GT`, `LTE`, or `GTE`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("EQ", "LT", "GT", "LTE", "GTE")},
			},
			"threshold": schema.StringAttribute{
				Required:      true,
				Description:   "Threshold value paired with the operator (e.g. `10.0`).",
				PlanModifiers: reqReplaceStr(),
			},
			"duration": schema.StringAttribute{
				Optional:      true,
				Description:   "Duration the condition must hold: `TWO`, `SIX`, `HOURLY`, or `DAILY`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("TWO", "SIX", "HOURLY", "DAILY")},
			},
			"notifications": schema.ListAttribute{
				Required:      true,
				ElementType:   types.StringType,
				Description:   "Notification channels: `EMAIL`, `SMS`, `PAGERDUTY`.",
				PlanModifiers: []planmodifier.List{listRequiresReplace()},
			},
		},
	}
}

func (r *alertRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *alertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	notifications, d := stringsFromList(ctx, plan.Notifications)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	rule, err := r.client.CreateAlertRule(ctx, client.CreateAlertRuleInput{
		DBType:        db,
		ClusterID:     plan.ClusterID.ValueString(),
		Type:          plan.Type.ValueString(),
		Metric:        stringValue(plan.Metric),
		Operator:      plan.Operator.ValueString(),
		Threshold:     plan.Threshold.ValueString(),
		Notifications: notifications,
		Duration:      stringValue(plan.Duration),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating alert rule", err.Error())
		return
	}
	plan.ID = types.StringValue(rule.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	rule, err := r.client.GetAlertRule(ctx, db, state.ClusterID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading alert rule", err.Error())
		return
	}
	if rule.Type != "" {
		state.Type = types.StringValue(rule.Type)
	}
	if rule.Metric != "" {
		state.Metric = types.StringValue(rule.Metric)
	}
	if rule.Operator != "" {
		state.Operator = types.StringValue(rule.Operator)
	}
	if rule.Threshold != "" {
		state.Threshold = types.StringValue(rule.Threshold)
	}
	if rule.AverageType != "" {
		state.Duration = types.StringValue(rule.AverageType)
	}
	if len(rule.Notifications) > 0 {
		list, d := stringsToList(ctx, rule.Notifications)
		resp.Diagnostics.Append(d...)
		state.Notifications = list
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is unreachable: every attribute forces replacement. It mirrors plan to
// state to satisfy the interface.
func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertRuleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertRuleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAlertRule(ctx, state.ID.ValueString(), true); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting alert rule", err.Error())
	}
}

// ImportState accepts "<database>:<cluster_id>:<rule_id>".
func (r *alertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := splitN(req.ID, 3)
	if parts == nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected \"database:cluster_id:rule_id\".")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[2])...)
}
