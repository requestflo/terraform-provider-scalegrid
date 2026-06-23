package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource              = (*cloudProfileResource)(nil)
	_ resource.ResourceWithConfigure = (*cloudProfileResource)(nil)
)

// NewCloudProfileResource is the constructor registered with the provider.
func NewCloudProfileResource() resource.Resource { return &cloudProfileResource{} }

type cloudProfileResource struct {
	client *client.Client
}

type cloudProfileResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Database           types.String `tfsdk:"database"`
	Region             types.String `tfsdk:"region"`
	AccessKey          types.String `tfsdk:"access_key"`
	SecretKey          types.String `tfsdk:"secret_key"`
	VPCID              types.String `tfsdk:"vpc_id"`
	SubnetID           types.String `tfsdk:"subnet_id"`
	VPCCIDR            types.String `tfsdk:"vpc_cidr"`
	SubnetCIDR         types.String `tfsdk:"subnet_cidr"`
	SecurityGroupID    types.String `tfsdk:"security_group_id"`
	SecurityGroupName  types.String `tfsdk:"security_group_name"`
	ConnectivityConfig types.String `tfsdk:"connectivity_config"`
	EnableSSH          types.Bool   `tfsdk:"enable_ssh"`
	CloudType          types.String `tfsdk:"cloud_type"`
}

func (r *cloudProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_profile"
}

func (r *cloudProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an AWS (EC2/VPC) ScaleGrid cloud profile for Bring Your Own Cloud " +
			"deployments. Azure cloud profiles require an interactive permission-granting script and " +
			"are not supported by this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Machine pool ID of the cloud profile.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Unique name of the cloud profile.",
				PlanModifiers: reqReplaceStr(),
			},
			"database": schema.StringAttribute{
				Required:      true,
				Description:   "Database engine this profile is for: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"region": schema.StringAttribute{
				Required:      true,
				Description:   "AWS region (e.g. `us-east-1`).",
				PlanModifiers: reqReplaceStr(),
			},
			"access_key": schema.StringAttribute{
				Required:    true,
				Description: "AWS access key. Can be updated in place to rotate credentials.",
			},
			"secret_key": schema.StringAttribute{
				Required:    true,
				Sensitive:   true,
				Description: "AWS secret key. Can be updated in place to rotate credentials.",
			},
			"vpc_id":              schema.StringAttribute{Required: true, Description: "AWS VPC ID.", PlanModifiers: reqReplaceStr()},
			"subnet_id":           schema.StringAttribute{Required: true, Description: "AWS subnet ID.", PlanModifiers: reqReplaceStr()},
			"vpc_cidr":            schema.StringAttribute{Required: true, Description: "VPC CIDR block.", PlanModifiers: reqReplaceStr()},
			"subnet_cidr":         schema.StringAttribute{Required: true, Description: "Subnet CIDR block.", PlanModifiers: reqReplaceStr()},
			"security_group_id":   schema.StringAttribute{Required: true, Description: "Security group ID.", PlanModifiers: reqReplaceStr()},
			"security_group_name": schema.StringAttribute{Required: true, Description: "Security group name.", PlanModifiers: reqReplaceStr()},
			"connectivity_config": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("INTERNET"),
				Description:   "Connectivity configuration: `INTERNET`, `INTRANET`, `SECURITYGROUP`, or `CUSTOMIPRANGE`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("INTERNET", "INTRANET", "SECURITYGROUP", "CUSTOMIPRANGE")},
			},
			"enable_ssh": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable SSH access to the VPC.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},
			"cloud_type": schema.StringAttribute{Computed: true, Description: "Cloud provider (e.g. AWS)."},
		},
	}
}

func (r *cloudProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *cloudProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan cloudProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	id, actionID, err := r.client.CreateAWSCloudProfile(ctx, client.CreateAWSCloudProfileInput{
		DBType:             db,
		Name:               plan.Name.ValueString(),
		Region:             plan.Region.ValueString(),
		AccessKey:          plan.AccessKey.ValueString(),
		SecretKey:          plan.SecretKey.ValueString(),
		VPCID:              plan.VPCID.ValueString(),
		SubnetID:           plan.SubnetID.ValueString(),
		VPCCIDR:            plan.VPCCIDR.ValueString(),
		SubnetCIDR:         plan.SubnetCIDR.ValueString(),
		SecurityGroupID:    plan.SecurityGroupID.ValueString(),
		SecurityGroupName:  plan.SecurityGroupName.ValueString(),
		ConnectivityConfig: stringValue(plan.ConnectivityConfig),
		EnableSSH:          plan.EnableSSH.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating cloud profile", err.Error())
		return
	}
	plan.ID = types.StringValue(id)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)

	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for cloud profile creation", err.Error())
		return
	}

	r.readInto(ctx, id, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state cloudProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	profile, err := r.client.GetCloudProfile(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cloud profile", err.Error())
		return
	}
	state.Name = types.StringValue(profile.Name)
	state.CloudType = optionalString(profile.CloudType())
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *cloudProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state cloudProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.AccessKey.ValueString() != state.AccessKey.ValueString() ||
		plan.SecretKey.ValueString() != state.SecretKey.ValueString() {
		if err := r.client.UpdateAWSCloudProfileKeys(ctx, state.ID.ValueString(),
			plan.AccessKey.ValueString(), plan.SecretKey.ValueString()); err != nil {
			resp.Diagnostics.AddError("Error updating cloud profile keys", err.Error())
			return
		}
	}

	plan.ID = state.ID
	r.readInto(ctx, state.ID.ValueString(), &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *cloudProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state cloudProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	actionID, err := r.client.DeleteCloudProfile(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting cloud profile", err.Error())
		return
	}
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for cloud profile deletion", err.Error())
	}
}

func (r *cloudProfileResource) readInto(ctx context.Context, id string, model *cloudProfileResourceModel, diags *diag.Diagnostics) {
	profile, err := r.client.GetCloudProfile(ctx, id)
	if err != nil {
		diags.AddError("Error reading cloud profile", err.Error())
		return
	}
	model.Name = types.StringValue(profile.Name)
	model.CloudType = optionalString(profile.CloudType())
}
