package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource              = (*backupScheduleResource)(nil)
	_ resource.ResourceWithConfigure = (*backupScheduleResource)(nil)
)

// NewBackupScheduleResource is the constructor registered with the provider.
func NewBackupScheduleResource() resource.Resource { return &backupScheduleResource{} }

type backupScheduleResource struct {
	client *client.Client
}

type backupScheduleResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Database       types.String `tfsdk:"database"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	Enabled        types.Bool   `tfsdk:"enabled"`
	IntervalHours  types.Int64  `tfsdk:"interval_hours"`
	Hour           types.Int64  `tfsdk:"hour"`
	RetentionLimit types.Int64  `tfsdk:"retention_limit"`
	Target         types.String `tfsdk:"target"`
}

func (r *backupScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_schedule"
}

func (r *backupScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Configures the scheduled (automated) backup policy for a ScaleGrid cluster. " +
			"Note: the ScaleGrid API does not expose a way to read back the current schedule, so this " +
			"resource is write-only — it applies the configured policy but cannot detect drift made " +
			"outside Terraform, and it does not support import.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier of the schedule (the cluster ID).",
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
				Description:   "ID of the cluster whose backup schedule is managed.",
				PlanModifiers: reqReplaceStr(),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether scheduled backups are enabled. Set to `false` to disable scheduling.",
			},
			"interval_hours": schema.Int64Attribute{
				Optional:    true,
				Description: "How often a backup runs, in hours: one of 1, 3, 6, 12, or 24. Required when `enabled` is true.",
				Validators:  []validator.Int64{int64validator.OneOf(1, 3, 6, 12, 24)},
			},
			"hour": schema.Int64Attribute{
				Optional:    true,
				Description: "Hour of day (0–23, UTC) at which the daily backup window starts.",
				Validators:  []validator.Int64{int64validator.Between(0, 23)},
			},
			"retention_limit": schema.Int64Attribute{
				Optional:    true,
				Description: "Maximum number of scheduled backups to retain.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"target": schema.StringAttribute{
				Optional: true,
				Description: "For replica sets, which node to back up: `PRIMARY`/`SECONDARY` (MongoDB), " +
					"`MASTER`/`SLAVE` (Redis/MySQL), or `MASTER`/`STANDBY` (PostgreSQL).",
			},
		},
	}
}

func (r *backupScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

// apply pushes the planned schedule to the API. Shared by Create and Update.
func (r *backupScheduleResource) apply(ctx context.Context, plan backupScheduleResourceModel, diags *diag.Diagnostics) {
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), diags)
	if !ok {
		return
	}
	enabled := plan.Enabled.ValueBool()
	if enabled && plan.IntervalHours.IsNull() {
		diags.AddError("Missing interval_hours",
			"interval_hours is required when enabled is true (one of 1, 3, 6, 12, or 24).")
		return
	}
	err := r.client.SetBackupSchedule(ctx, db, plan.ClusterID.ValueString(), enabled,
		int(plan.IntervalHours.ValueInt64()), int(plan.Hour.ValueInt64()),
		int(plan.RetentionLimit.ValueInt64()), stringValue(plan.Target))
	if err != nil {
		diags.AddError("Error setting backup schedule", err.Error())
	}
}

func (r *backupScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = plan.ClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read is a no-op: the API offers no endpoint to fetch the current schedule, so
// the configured state is preserved as-is (drift cannot be detected).
func (r *backupScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state backupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *backupScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan backupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.apply(ctx, plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = plan.ClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete disables scheduled backups on the cluster.
func (r *backupScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state backupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.SetBackupSchedule(ctx, db, state.ClusterID.ValueString(), false, 0, 0, 0, ""); err != nil {
		resp.Diagnostics.AddError("Error disabling backup schedule", err.Error())
	}
}
