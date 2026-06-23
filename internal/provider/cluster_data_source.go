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
	_ datasource.DataSource              = (*clusterDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*clusterDataSource)(nil)
)

// NewClusterDataSource is the constructor registered with the provider.
func NewClusterDataSource() datasource.DataSource { return &clusterDataSource{} }

type clusterDataSource struct {
	client *client.Client
}

type clusterDataSourceModel struct {
	Database          types.String `tfsdk:"database"`
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Status            types.String `tfsdk:"status"`
	Size              types.String `tfsdk:"size"`
	Version           types.String `tfsdk:"version"`
	ClusterType       types.String `tfsdk:"cluster_type"`
	DiskSizeGB        types.Int64  `tfsdk:"disk_size_gb"`
	SSLEnabled        types.Bool   `tfsdk:"ssl_enabled"`
	EncryptionEnabled types.Bool   `tfsdk:"encryption_enabled"`
}

func (d *clusterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (d *clusterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single ScaleGrid cluster by ID or name.",
		Attributes: map[string]schema.Attribute{
			"database": schema.StringAttribute{
				Required:    true,
				Description: "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				Validators:  []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"id":   schema.StringAttribute{Optional: true, Computed: true, Description: "Cluster ID. Either `id` or `name` must be set."},
			"name": schema.StringAttribute{Optional: true, Computed: true, Description: "Cluster name. Either `id` or `name` must be set."},

			"status":             schema.StringAttribute{Computed: true, Description: "Lifecycle status."},
			"size":               schema.StringAttribute{Computed: true, Description: "Instance size tier."},
			"version":            schema.StringAttribute{Computed: true, Description: "Engine version."},
			"cluster_type":       schema.StringAttribute{Computed: true, Description: "Topology."},
			"disk_size_gb":       schema.Int64Attribute{Computed: true, Description: "Disk size in GB."},
			"ssl_enabled":        schema.BoolAttribute{Computed: true, Description: "Whether SSL is enabled."},
			"encryption_enabled": schema.BoolAttribute{Computed: true, Description: "Whether encryption at rest is enabled."},
		},
	}
}

func (d *clusterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	d.client = c
}

func (d *clusterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config clusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(config.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	id := stringValue(config.ID)
	name := stringValue(config.Name)
	if id == "" && name == "" {
		resp.Diagnostics.AddError("Missing lookup key", "One of `id` or `name` must be set.")
		return
	}

	var cluster *client.Cluster
	var err error
	if id != "" {
		cluster, err = d.client.GetCluster(ctx, db, id)
	} else {
		cluster, err = d.client.FindClusterByName(ctx, db, name)
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster", err.Error())
		return
	}

	config.ID = types.StringValue(cluster.ID)
	config.Name = types.StringValue(cluster.Name)
	config.Status = optionalString(cluster.Status)
	config.Size = optionalString(cluster.Size)
	config.Version = optionalString(cluster.VersionStr)
	config.ClusterType = optionalString(cluster.ClusterType)
	config.DiskSizeGB = types.Int64Value(cluster.DiskSizeGB)
	config.SSLEnabled = types.BoolValue(cluster.SSLEnabled)
	config.EncryptionEnabled = types.BoolValue(cluster.EncryptionEnabled)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
