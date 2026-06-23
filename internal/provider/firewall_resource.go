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
	_ resource.Resource                = (*firewallResource)(nil)
	_ resource.ResourceWithConfigure   = (*firewallResource)(nil)
	_ resource.ResourceWithImportState = (*firewallResource)(nil)
)

// NewFirewallResource is the constructor registered with the provider.
func NewFirewallResource() resource.Resource { return &firewallResource{} }

type firewallResource struct {
	client *client.Client
}

type firewallResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Database  types.String `tfsdk:"database"`
	ClusterID types.String `tfsdk:"cluster_id"`
	CIDRList  types.List   `tfsdk:"cidr_list"`
}

func (r *firewallResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_firewall"
}

func (r *firewallResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the cluster-level IP whitelist (firewall) for a ScaleGrid cluster. This " +
			"resource owns the complete CIDR list for the cluster; applying it replaces any existing rules.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier of the firewall configuration (the cluster ID).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database": schema.StringAttribute{
				Required:      true,
				Description:   "Database engine of the cluster: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"cluster_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the cluster the whitelist applies to.",
				PlanModifiers: reqReplaceStr(),
			},
			"cidr_list": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Complete list of CIDR ranges allowed to connect (e.g. `[\"10.0.0.0/8\"]`).",
			},
		},
	}
}

func (r *firewallResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *firewallResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan firewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	cidrs, d := stringsFromList(ctx, plan.CIDRList)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.SetFirewallRules(ctx, db, plan.ClusterID.ValueString(), cidrs); err != nil {
		resp.Diagnostics.AddError("Error setting firewall rules", err.Error())
		return
	}
	plan.ID = plan.ClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *firewallResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state firewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	cidrs, err := r.client.GetFirewallRules(ctx, db, state.ClusterID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading firewall rules", err.Error())
		return
	}
	list, d := stringsToList(ctx, cidrs)
	resp.Diagnostics.Append(d...)
	state.CIDRList = list
	state.ID = state.ClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *firewallResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan firewallResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	cidrs, d := stringsFromList(ctx, plan.CIDRList)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.SetFirewallRules(ctx, db, plan.ClusterID.ValueString(), cidrs); err != nil {
		resp.Diagnostics.AddError("Error updating firewall rules", err.Error())
		return
	}
	plan.ID = plan.ClusterID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete clears the whitelist (removes all rules).
func (r *firewallResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state firewallResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	if err := r.client.SetFirewallRules(ctx, db, state.ClusterID.ValueString(), []string{}); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error clearing firewall rules", err.Error())
	}
}

// ImportState accepts "<database>:<cluster_id>".
func (r *firewallResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	db, id, ok := splitImportID(req.ID)
	if !ok {
		resp.Diagnostics.AddError("Invalid import ID", "Expected \"database:cluster_id\".")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database"), db)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
