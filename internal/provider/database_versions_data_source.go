package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ datasource.DataSource              = (*databaseVersionsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*databaseVersionsDataSource)(nil)
)

// NewDatabaseVersionsDataSource is the constructor registered with the provider.
func NewDatabaseVersionsDataSource() datasource.DataSource { return &databaseVersionsDataSource{} }

type databaseVersionsDataSource struct {
	client *client.Client
}

type databaseVersionsDataSourceModel struct {
	Database      types.String `tfsdk:"database"`
	CloudProvider types.String `tfsdk:"cloud_provider"`
	Versions      types.Map    `tfsdk:"versions"`
}

func (d *databaseVersionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_versions"
}

func (d *databaseVersionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Returns the database engine versions available for a given engine and cloud provider.",
		Attributes: map[string]schema.Attribute{
			"database": schema.StringAttribute{
				Required:    true,
				Description: "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				Validators:  []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"cloud_provider": schema.StringAttribute{
				Required:    true,
				Description: "Cloud provider: `AWS`, `AZURE`, or `DO`.",
				Validators:  []validator.String{stringvalidator.OneOf("AWS", "AZURE", "DO")},
			},
			"versions": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Map of version identifier to display name.",
			},
		},
	}
}

func (d *databaseVersionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	d.client = c
}

func (d *databaseVersionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config databaseVersionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(config.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	versions, err := d.client.GetDatabaseVersions(ctx, db, config.CloudProvider.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error fetching database versions", err.Error())
		return
	}
	mapVal, diags := types.MapValueFrom(ctx, types.StringType, versions)
	resp.Diagnostics.Append(diags...)
	config.Versions = mapVal
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
