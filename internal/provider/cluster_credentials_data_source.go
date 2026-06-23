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
	_ datasource.DataSource              = (*clusterCredentialsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*clusterCredentialsDataSource)(nil)
)

// NewClusterCredentialsDataSource is the constructor registered with the provider.
func NewClusterCredentialsDataSource() datasource.DataSource { return &clusterCredentialsDataSource{} }

type clusterCredentialsDataSource struct {
	client *client.Client
}

type connectionStringModel struct {
	Driver           types.String `tfsdk:"driver"`
	ConnectionString types.String `tfsdk:"connection_string"`
}

type clusterCredentialsDataSourceModel struct {
	Database          types.String            `tfsdk:"database"`
	ClusterID         types.String            `tfsdk:"cluster_id"`
	Username          types.String            `tfsdk:"username"`
	Password          types.String            `tfsdk:"password"`
	CommandLine       types.String            `tfsdk:"command_line"`
	ConnectionStrings []connectionStringModel `tfsdk:"connection_strings"`
}

func (d *clusterCredentialsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster_credentials"
}

func (d *clusterCredentialsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches the root database credentials and connection strings for a cluster.",
		Attributes: map[string]schema.Attribute{
			"database": schema.StringAttribute{
				Required:    true,
				Description: "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				Validators:  []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"cluster_id":   schema.StringAttribute{Required: true, Description: "ID of the cluster."},
			"username":     schema.StringAttribute{Computed: true, Description: "Root database username."},
			"password":     schema.StringAttribute{Computed: true, Sensitive: true, Description: "Root database password."},
			"command_line": schema.StringAttribute{Computed: true, Sensitive: true, Description: "Command-line connection syntax."},
			"connection_strings": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Driver-specific connection strings.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"driver":            schema.StringAttribute{Computed: true, Description: "Driver name."},
						"connection_string": schema.StringAttribute{Computed: true, Sensitive: true, Description: "Connection string."},
					},
				},
			},
		},
	}
}

func (d *clusterCredentialsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	d.client = c
}

func (d *clusterCredentialsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config clusterCredentialsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(config.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	creds, err := d.client.GetCredentials(ctx, db, config.ClusterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error fetching cluster credentials", err.Error())
		return
	}

	config.Username = optionalString(creds.User)
	config.Password = optionalString(creds.Password)
	config.CommandLine = optionalString(creds.CommandLine)
	config.ConnectionStrings = make([]connectionStringModel, 0, len(creds.ConnectionStrings))
	for _, cs := range creds.ConnectionStrings {
		config.ConnectionStrings = append(config.ConnectionStrings, connectionStringModel{
			Driver:           optionalString(cs.Driver),
			ConnectionString: optionalString(cs.ConnStr),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
