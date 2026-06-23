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
	_ datasource.DataSource              = (*clustersDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*clustersDataSource)(nil)
)

// NewClustersDataSource is the constructor registered with the provider.
func NewClustersDataSource() datasource.DataSource { return &clustersDataSource{} }

type clustersDataSource struct {
	client *client.Client
}

type clusterListItemModel struct {
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

type clustersDataSourceModel struct {
	Database types.String           `tfsdk:"database"`
	Clusters []clusterListItemModel `tfsdk:"clusters"`
}

func (d *clustersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clusters"
}

func (d *clustersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all ScaleGrid clusters of a given database engine.",
		Attributes: map[string]schema.Attribute{
			"database": schema.StringAttribute{
				Required:    true,
				Description: "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				Validators:  []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"clusters": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The matching clusters.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true, Description: "Cluster ID."},
						"name":               schema.StringAttribute{Computed: true, Description: "Cluster name."},
						"status":             schema.StringAttribute{Computed: true, Description: "Lifecycle status."},
						"size":               schema.StringAttribute{Computed: true, Description: "Instance size tier."},
						"version":            schema.StringAttribute{Computed: true, Description: "Engine version."},
						"cluster_type":       schema.StringAttribute{Computed: true, Description: "Topology."},
						"disk_size_gb":       schema.Int64Attribute{Computed: true, Description: "Disk size in GB."},
						"ssl_enabled":        schema.BoolAttribute{Computed: true, Description: "Whether SSL is enabled."},
						"encryption_enabled": schema.BoolAttribute{Computed: true, Description: "Whether encryption at rest is enabled."},
					},
				},
			},
		},
	}
}

func (d *clustersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	d.client = c
}

func (d *clustersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config clustersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(config.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	clusters, err := d.client.ListClusters(ctx, db)
	if err != nil {
		resp.Diagnostics.AddError("Error listing clusters", err.Error())
		return
	}

	config.Clusters = make([]clusterListItemModel, 0, len(clusters))
	for i := range clusters {
		c := clusters[i]
		config.Clusters = append(config.Clusters, clusterListItemModel{
			ID:                types.StringValue(c.ID),
			Name:              types.StringValue(c.Name),
			Status:            optionalString(c.Status),
			Size:              optionalString(c.Size),
			Version:           optionalString(c.VersionStr),
			ClusterType:       optionalString(c.ClusterType),
			DiskSizeGB:        types.Int64Value(c.DiskSizeGB),
			SSLEnabled:        types.BoolValue(c.SSLEnabled),
			EncryptionEnabled: types.BoolValue(c.EncryptionEnabled),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
