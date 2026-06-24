package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

var (
	_ resource.Resource              = (*redisClusterResource)(nil)
	_ resource.ResourceWithConfigure = (*redisClusterResource)(nil)
)

// NewRedisClusterResource is the constructor registered with the provider.
func NewRedisClusterResource() resource.Resource { return &redisClusterResource{} }

type redisClusterResource struct {
	client *client.Client
}

type redisClusterModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Size              types.String `tfsdk:"size"`
	Version           types.String `tfsdk:"version"`
	CloudProfileNames types.List   `tfsdk:"cloud_profile_names"`
	Region            types.String `tfsdk:"region"`
	ShardCount        types.Int64  `tfsdk:"shard_count"`
	EncryptDisk       types.Bool   `tfsdk:"encrypt_disk"`
	EnableSSL         types.Bool   `tfsdk:"enable_ssl"`
	Paused            types.Bool   `tfsdk:"paused"`

	ServerCount         types.Int64 `tfsdk:"server_count"`
	SentinelCount       types.Int64 `tfsdk:"sentinel_count"`
	ClusterMode         types.Bool  `tfsdk:"cluster_mode"`
	BackupIntervalHours types.Int64 `tfsdk:"backup_interval_hours"`

	Status            types.String `tfsdk:"status"`
	ClusterType       types.String `tfsdk:"cluster_type"`
	DiskSizeGB        types.Int64  `tfsdk:"disk_size_gb"`
	EncryptionEnabled types.Bool   `tfsdk:"encryption_enabled"`
	SSLActive         types.Bool   `tfsdk:"ssl_active"`
}

func (r *redisClusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_cluster"
}

func (r *redisClusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ScaleGrid Redis™ deployment. `size` and `paused` are applied in place; " +
			"all other attributes force replacement.",
		Attributes: mergeAttributes(map[string]schema.Attribute{
			"server_count": schema.Int64Attribute{
				Optional:      true,
				Description:   "Nodes per shard. 1 for standalone.",
				PlanModifiers: reqReplaceInt(),
			},
			"sentinel_count": schema.Int64Attribute{
				Optional:      true,
				Description:   "Number of sentinel nodes (master/slave deployments).",
				PlanModifiers: reqReplaceInt(),
			},
			"cluster_mode": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable Redis cluster mode (requires 3 or 4 shards).",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},
			"backup_interval_hours": schema.Int64Attribute{
				Optional:      true,
				Description:   "Scheduled backup interval in hours, one of 1, 3, 6, 12, or 24 (0 disables).",
				PlanModifiers: reqReplaceInt(),
			},
		}),
	}
}

func (r *redisClusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *redisClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan redisClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	profileNames, d := stringsFromList(ctx, plan.CloudProfileNames)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	nodeCount := nodesPerCluster(int(plan.ShardCount.ValueInt64()), int(plan.ServerCount.ValueInt64()))
	poolIDs, err := resolveMachinePools(ctx, r.client, client.DBRedis, profileNames, plan.Region.ValueString(), nodeCount)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving cloud profiles", err.Error())
		return
	}

	in := client.CreateClusterInput{
		DBType:                client.DBRedis,
		Name:                  plan.Name.ValueString(),
		Size:                  plan.Size.ValueString(),
		Version:               plan.Version.ValueString(),
		ShardCount:            int(plan.ShardCount.ValueInt64()),
		ServerCount:           int(plan.ServerCount.ValueInt64()),
		SentinelCount:         int(plan.SentinelCount.ValueInt64()),
		MachinePoolIDs:        poolIDs,
		EncryptDisk:           plan.EncryptDisk.ValueBool(),
		ClusterMode:           plan.ClusterMode.ValueBool(),
		BackupIntervalInHours: int(plan.BackupIntervalHours.ValueInt64()),
	}

	clusterID, actionID, err := r.client.CreateCluster(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", err.Error())
		return
	}
	plan.ID = types.StringValue(clusterID)
	persistIDEarly(ctx, resp, clusterID)

	tflog.Info(ctx, "waiting for Redis cluster provisioning", map[string]any{"cluster_id": clusterID, "action_id": actionID})
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for cluster provisioning", err.Error())
		return
	}
	if plan.Paused.ValueBool() {
		if _, err := r.client.PauseCluster(ctx, client.DBRedis, clusterID); err != nil {
			resp.Diagnostics.AddError("Error pausing cluster after creation", err.Error())
			return
		}
	}

	cluster, err := r.client.GetCluster(ctx, client.DBRedis, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after creation", err.Error())
		return
	}
	plan.applyComputed(cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redisClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state redisClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cluster, err := r.client.GetCluster(ctx, client.DBRedis, state.ID.ValueString())
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

func (r *redisClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state redisClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := scaleAndPause(ctx, r.client, client.DBRedis, id,
		plan.Size.ValueString(), state.Size.ValueString(),
		plan.Paused.ValueBool(), state.Paused.ValueBool()); err != nil {
		resp.Diagnostics.AddError("Error updating cluster", err.Error())
		return
	}
	cluster, err := r.client.GetCluster(ctx, client.DBRedis, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after update", err.Error())
		return
	}
	plan.ID = state.ID
	plan.applyComputed(cluster)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *redisClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state redisClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteCluster(ctx, r.client, client.DBRedis, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting cluster", err.Error())
	}
}

func (m *redisClusterModel) applyComputed(cluster *client.Cluster) {
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
