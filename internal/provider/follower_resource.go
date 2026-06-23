package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource              = (*followerResource)(nil)
	_ resource.ResourceWithConfigure = (*followerResource)(nil)
)

// NewFollowerResource is the constructor registered with the provider.
func NewFollowerResource() resource.Resource { return &followerResource{} }

type followerResource struct {
	client *client.Client
}

type followerResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Database        types.String `tfsdk:"database"`
	TargetClusterID types.String `tfsdk:"target_cluster_id"`
	SourceClusterID types.String `tfsdk:"source_cluster_id"`
	IntervalHours   types.Int64  `tfsdk:"interval_hours"`
	StartHour       types.Int64  `tfsdk:"start_hour"`
}

func (r *followerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_follower"
}

func (r *followerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a follower relationship in which one cluster periodically syncs from another.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier of the follower relationship (the target cluster ID).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database": schema.StringAttribute{
				Required:      true,
				Description:   "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"target_cluster_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the follower (destination) cluster.",
				PlanModifiers: reqReplaceStr(),
			},
			"source_cluster_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the source cluster being followed.",
				PlanModifiers: reqReplaceStr(),
			},
			"interval_hours": schema.Int64Attribute{
				Required:      true,
				Description:   "Hours between each sync from the source cluster.",
				PlanModifiers: reqReplaceInt(),
				Validators:    []validator.Int64{int64validator.AtLeast(1)},
			},
			"start_hour": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Default:       int64default.StaticInt64(0),
				Description:   "Hour of day (0-23) at which the first sync starts.",
				PlanModifiers: reqReplaceInt(),
				Validators:    []validator.Int64{int64validator.Between(0, 23)},
			},
		},
	}
}

func (r *followerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *followerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan followerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.CreateFollower(ctx, db, plan.TargetClusterID.ValueString(),
		plan.SourceClusterID.ValueString(), int(plan.IntervalHours.ValueInt64()), int(plan.StartHour.ValueInt64())); err != nil {
		resp.Diagnostics.AddError("Error creating follower relationship", err.Error())
		return
	}
	plan.ID = plan.TargetClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *followerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state followerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	status, err := r.client.GetFollowerStatus(ctx, db, state.TargetClusterID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading follower relationship", err.Error())
		return
	}
	if status.SourceCluster.ID != "" {
		state.SourceClusterID = types.StringValue(status.SourceCluster.ID)
	}
	if status.SyncSchedule.IntervalInHours > 0 {
		state.IntervalHours = types.Int64Value(int64(status.SyncSchedule.IntervalInHours))
	}
	state.ID = state.TargetClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update mirrors plan to state; all attributes force replacement.
func (r *followerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan followerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *followerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state followerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.BreakFollower(ctx, db, state.TargetClusterID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error breaking follower relationship", err.Error())
	}
}
