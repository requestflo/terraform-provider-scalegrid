package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ datasource.DataSource              = (*cloudProfileDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*cloudProfileDataSource)(nil)
)

// NewCloudProfileDataSource is the constructor registered with the provider.
func NewCloudProfileDataSource() datasource.DataSource { return &cloudProfileDataSource{} }

type cloudProfileDataSource struct {
	client *client.Client
}

type cloudProfileDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	CloudType types.String `tfsdk:"cloud_type"`
	Database  types.String `tfsdk:"database"`
	Status    types.String `tfsdk:"status"`
	Shared    types.Bool   `tfsdk:"shared"`
}

func (d *cloudProfileDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cloud_profile"
}

func (d *cloudProfileDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single ScaleGrid cloud profile by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":         schema.StringAttribute{Optional: true, Computed: true, Description: "Cloud profile (machine pool) ID. Either `id` or `name` must be set."},
			"name":       schema.StringAttribute{Optional: true, Computed: true, Description: "Cloud profile name. Either `id` or `name` must be set."},
			"cloud_type": schema.StringAttribute{Computed: true, Description: "Cloud provider (e.g. AWS)."},
			"database":   schema.StringAttribute{Computed: true, Description: "Database engine the profile is for."},
			"status":     schema.StringAttribute{Computed: true, Description: "Status of the cloud profile."},
			"shared":     schema.BoolAttribute{Computed: true, Description: "Whether this is a shared (Dedicated plan) profile."},
		},
	}
}

func (d *cloudProfileDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	d.client = c
}

func (d *cloudProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config cloudProfileDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := stringValue(config.ID)
	name := stringValue(config.Name)
	if id == "" && name == "" {
		resp.Diagnostics.AddError("Missing lookup key", "One of `id` or `name` must be set.")
		return
	}

	var profile *client.CloudProfile
	var err error
	if id != "" {
		profile, err = d.client.GetCloudProfile(ctx, id)
	} else {
		profile, err = d.client.FindCloudProfileByName(ctx, name)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading cloud profile", err.Error())
		return
	}

	config.ID = types.StringValue(profile.ID)
	config.Name = types.StringValue(profile.Name)
	config.CloudType = optionalString(profile.CloudType())
	config.Database = optionalString(profile.DBType)
	config.Status = optionalString(profile.Status)
	config.Shared = types.BoolValue(profile.Shared)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
