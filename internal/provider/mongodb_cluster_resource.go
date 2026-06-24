package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource              = (*mongoClusterResource)(nil)
	_ resource.ResourceWithConfigure = (*mongoClusterResource)(nil)
)

// NewMongoClusterResource is the constructor registered with the provider.
func NewMongoClusterResource() resource.Resource { return &mongoClusterResource{} }

type mongoClusterResource struct {
	client *client.Client
}

type mongoClusterModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Size              types.String `tfsdk:"size"`
	Version           types.String `tfsdk:"version"`
	CloudProfileNames types.List   `tfsdk:"cloud_profile_names"`
	ShardCount        types.Int64  `tfsdk:"shard_count"`
	EncryptDisk       types.Bool   `tfsdk:"encrypt_disk"`
	EnableSSL         types.Bool   `tfsdk:"enable_ssl"`
	Paused            types.Bool   `tfsdk:"paused"`

	ReplicaCount    types.Int64  `tfsdk:"replica_count"`
	MongoEngine     types.String `tfsdk:"mongo_engine"`
	CompressionAlgo types.String `tfsdk:"compression_algo"`

	Status            types.String `tfsdk:"status"`
	ClusterType       types.String `tfsdk:"cluster_type"`
	DiskSizeGB        types.Int64  `tfsdk:"disk_size_gb"`
	EncryptionEnabled types.Bool   `tfsdk:"encryption_enabled"`
	SSLActive         types.Bool   `tfsdk:"ssl_active"`
}

func (r *mongoClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mongodb_cluster"
}

func (r *mongoClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ScaleGrid MongoDB deployment. `size` and `paused` are applied in place; " +
			"all other attributes force replacement.",
		Attributes: mergeAttributes(map[string]schema.Attribute{
			"replica_count": schema.Int64Attribute{
				Optional:      true,
				Description:   "Nodes per shard. 1 for standalone, more for a replica set.",
				PlanModifiers: reqReplaceInt(),
			},
			"mongo_engine": schema.StringAttribute{
				Optional:      true,
				Description:   "Storage engine: `wiredtiger` (default) or `mmapv1`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("wiredtiger", "mmapv1")},
			},
			"compression_algo": schema.StringAttribute{
				Optional:      true,
				Description:   "Compression algorithm: `snappy` or `zlib`. Omit to leave data uncompressed.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("snappy", "zlib")},
			},
		}),
	}
}

func (r *mongoClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *mongoClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mongoClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileNames, d := stringsFromList(ctx, plan.CloudProfileNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	poolIDs, err := resolveProfiles(ctx, r.client, profileNames)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving cloud profiles", err.Error())
		return
	}

	in := client.CreateClusterInput{
		DBType:          client.DBMongo,
		Name:            plan.Name.ValueString(),
		Size:            plan.Size.ValueString(),
		Version:         plan.Version.ValueString(),
		ShardCount:      int(plan.ShardCount.ValueInt64()),
		ReplicaCount:    int(plan.ReplicaCount.ValueInt64()),
		MachinePoolIDs:  poolIDs,
		EncryptDisk:     plan.EncryptDisk.ValueBool(),
		EnableSSL:       plan.EnableSSL.ValueBool(),
		MongoEngine:     stringValue(plan.MongoEngine),
		CompressionAlgo: stringValue(plan.CompressionAlgo),
	}

	clusterID, actionID, err := r.client.CreateCluster(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", err.Error())
		return
	}
	plan.ID = types.StringValue(clusterID)
	persistIDEarly(ctx, resp, clusterID)

	tflog.Info(ctx, "waiting for MongoDB cluster provisioning", map[string]any{"cluster_id": clusterID, "action_id": actionID})
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for cluster provisioning", err.Error())
		return
	}
	if plan.Paused.ValueBool() {
		if _, err := r.client.PauseCluster(ctx, client.DBMongo, clusterID); err != nil {
			resp.Diagnostics.AddError("Error pausing cluster after creation", err.Error())
			return
		}
	}

	cluster, err := r.client.GetCluster(ctx, client.DBMongo, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after creation", err.Error())
		return
	}
	plan.applyComputed(cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mongoClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mongoClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cluster, err := r.client.GetCluster(ctx, client.DBMongo, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cluster", err.Error())
		return
	}
	state.applyComputed(cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mongoClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state mongoClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := scaleAndPause(ctx, r.client, client.DBMongo, id,
		plan.Size.ValueString(), state.Size.ValueString(),
		plan.Paused.ValueBool(), state.Paused.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error updating cluster", err.Error())
		return
	}
	cluster, err := r.client.GetCluster(ctx, client.DBMongo, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after update", err.Error())
		return
	}
	plan.ID = state.ID
	plan.applyComputed(cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mongoClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mongoClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteCluster(ctx, r.client, client.DBMongo, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting cluster", err.Error())
	}
}

func (m *mongoClusterModel) applyComputed(cluster *client.Cluster) {
	m.ID = types.StringValue(cluster.ID)
	cc := computedFromCluster(cluster)
	if !cc.Size.IsNull() {
		m.Size = cc.Size
	}
	m.Status = cc.Status
	m.ClusterType = cc.ClusterType
	m.DiskSizeGB = cc.DiskSizeGB
	m.EncryptionEnabled = cc.EncryptionEnabled
	m.SSLActive = cc.SSLActive
}
