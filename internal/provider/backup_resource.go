package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource              = (*backupResource)(nil)
	_ resource.ResourceWithConfigure = (*backupResource)(nil)
)

// NewBackupResource is the constructor registered with the provider.
func NewBackupResource() resource.Resource { return &backupResource{} }

type backupResource struct {
	client *client.Client
}

type backupResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Database  types.String `tfsdk:"database"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Name      types.String `tfsdk:"name"`
	Comment   types.String `tfsdk:"comment"`
	Target    types.String `tfsdk:"target"`
	Type      types.String `tfsdk:"type"`
	Created   types.String `tfsdk:"created"`
}

func (r *backupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup"
}

func (r *backupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers and manages an on-demand backup of a ScaleGrid cluster.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Unique identifier of the backup.",
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
				Description:   "ID of the cluster to back up.",
				PlanModifiers: reqReplaceStr(),
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Unique name for the backup.",
				PlanModifiers: reqReplaceStr(),
			},
			"comment": schema.StringAttribute{
				Optional:      true,
				Description:   "Optional comment describing the backup.",
				PlanModifiers: reqReplaceStr(),
			},
			"target": schema.StringAttribute{
				Optional: true,
				Description: "For replica sets, which node to back up: `PRIMARY`/`SECONDARY` (MongoDB), " +
					"`MASTER`/`SLAVE` (Redis/MySQL), or `MASTER`/`STANDBY` (PostgreSQL). PostgreSQL " +
					"requires a target and defaults to `MASTER`.",
				PlanModifiers: reqReplaceStr(),
			},
			"type":    schema.StringAttribute{Computed: true, Description: "Backup type (`ONDEMAND` or `SCHEDULED`)."},
			"created": schema.StringAttribute{Computed: true, Description: "Creation time as a Unix timestamp (seconds, UTC)."},
		},
	}
}

func (r *backupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *backupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan backupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	actionID, err := r.client.StartBackup(ctx, db, plan.ClusterID.ValueString(),
		plan.Name.ValueString(), stringValue(plan.Comment), stringValue(plan.Target))
	if err != nil {
		resp.Diagnostics.AddError("Error starting backup", err.Error())
		return
	}
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for backup", err.Error())
		return
	}

	backup, err := r.client.FindBackupByName(ctx, db, plan.ClusterID.ValueString(), plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading backup after creation", err.Error())
		return
	}
	r.mapComputed(backup, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state backupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	backup, err := r.client.FindBackupByName(ctx, db, state.ClusterID.ValueString(), state.Name.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading backup", err.Error())
		return
	}
	r.mapComputed(backup, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update mirrors plan to state; all configurable attributes force replacement.
func (r *backupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan backupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *backupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state backupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	actionID, err := r.client.DeleteBackup(ctx, db, state.ClusterID.ValueString(), state.ID.ValueString(), true)
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting backup", err.Error())
		return
	}
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for backup deletion", err.Error())
	}
}

func (r *backupResource) mapComputed(b *client.Backup, model *backupResourceModel) {
	model.ID = types.StringValue(b.ID)
	model.Type = optionalString(b.Type)
	model.Created = optionalString(b.Created)
	if b.Comment != "" {
		model.Comment = types.StringValue(b.Comment)
	}
}
