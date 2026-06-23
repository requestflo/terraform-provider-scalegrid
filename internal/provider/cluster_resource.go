package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

const clusterPollInterval = 20 * time.Second

var (
	_ resource.Resource              = (*clusterResource)(nil)
	_ resource.ResourceWithConfigure = (*clusterResource)(nil)
)

// NewClusterResource is the constructor registered with the provider.
func NewClusterResource() resource.Resource { return &clusterResource{} }

type clusterResource struct {
	client *client.Client
}

type clusterResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Database          types.String `tfsdk:"database"`
	Name              types.String `tfsdk:"name"`
	Size              types.String `tfsdk:"size"`
	Version           types.String `tfsdk:"version"`
	CloudProfileNames types.List   `tfsdk:"cloud_profile_names"`
	ShardCount        types.Int64  `tfsdk:"shard_count"`
	ReplicaCount      types.Int64  `tfsdk:"replica_count"`
	ServerCount       types.Int64  `tfsdk:"server_count"`
	SentinelCount     types.Int64  `tfsdk:"sentinel_count"`
	SentinelProfiles  types.List   `tfsdk:"sentinel_cloud_profile_names"`
	EncryptDisk       types.Bool   `tfsdk:"encrypt_disk"`
	EnableSSL         types.Bool   `tfsdk:"enable_ssl"`
	Paused            types.Bool   `tfsdk:"paused"`

	// MongoDB
	MongoEngine     types.String `tfsdk:"mongo_engine"`
	CompressionAlgo types.String `tfsdk:"compression_algo"`
	// Redis
	ClusterMode         types.Bool   `tfsdk:"cluster_mode"`
	BackupIntervalHours types.Int64  `tfsdk:"backup_interval_hours"`
	MaxMemoryPolicy     types.String `tfsdk:"maxmemory_policy"`
	EnableRDB           types.Bool   `tfsdk:"enable_rdb"`
	EnableAOF           types.Bool   `tfsdk:"enable_aof"`
	// MySQL
	ReplicaConfig types.Int64 `tfsdk:"replica_config"`
	// PostgreSQL
	ReplicationType types.String `tfsdk:"replication_type"`
	SyncCommitType  types.String `tfsdk:"sync_commit_type"`
	EnablePgBouncer types.Bool   `tfsdk:"enable_pgbouncer"`

	// Computed
	Status            types.String `tfsdk:"status"`
	ClusterType       types.String `tfsdk:"cluster_type"`
	DiskSizeGB        types.Int64  `tfsdk:"disk_size_gb"`
	EncryptionEnabled types.Bool   `tfsdk:"encryption_enabled"`
	SSLActive         types.Bool   `tfsdk:"ssl_active"`
}

func (r *clusterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func reqReplaceStr() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}
func reqReplaceInt() []planmodifier.Int64 {
	return []planmodifier.Int64{int64planmodifier.RequiresReplace()}
}

func (r *clusterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ScaleGrid database deployment (cluster). Most attributes are immutable " +
			"and changing them forces a new cluster; `size` and `paused` are applied in place.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Unique identifier of the cluster.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"database": schema.StringAttribute{
				Required:      true,
				Description:   "Database engine: `mongodb`, `redis`, `mysql`, or `postgresql`.",
				PlanModifiers: reqReplaceStr(),
				Validators:    []validator.String{stringvalidator.OneOf("mongodb", "redis", "mysql", "postgresql")},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Unique name of the cluster.",
				PlanModifiers: reqReplaceStr(),
			},
			"size": schema.StringAttribute{
				Required: true,
				Description: "Instance size tier: `Micro`, `Small`, `Medium`, `Large`, `XLarge`, " +
					"`X2XLarge`, or `X4XLarge`. Changing this scales the cluster in place.",
				Validators: []validator.String{stringvalidator.OneOf(client.ValidSizes...)},
			},
			"version": schema.StringAttribute{
				Required:      true,
				Description:   "Database engine version. Use the `scalegrid_database_versions` data source to discover valid values.",
				PlanModifiers: reqReplaceStr(),
			},
			"cloud_profile_names": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Names of the cloud profiles to deploy nodes into. The count should match " +
					"the number of nodes (replicas/servers) across shards.",
				PlanModifiers: []planmodifier.List{listRequiresReplace()},
			},
			"shard_count": schema.Int64Attribute{
				Optional:      true,
				Computed:      true,
				Default:       int64default.StaticInt64(1),
				Description:   "Number of shards. 1 for standalone/replica set; more for sharded. Redis cluster-mode requires 3 or 4.",
				PlanModifiers: reqReplaceInt(),
			},
			"replica_count": schema.Int64Attribute{
				Optional:      true,
				Description:   "Nodes per shard for MongoDB/MySQL/PostgreSQL. 1 for standalone, more for replica set.",
				PlanModifiers: reqReplaceInt(),
			},
			"server_count": schema.Int64Attribute{
				Optional:      true,
				Description:   "Nodes per shard for Redis. 1 for standalone.",
				PlanModifiers: reqReplaceInt(),
			},
			"sentinel_count": schema.Int64Attribute{
				Optional:      true,
				Description:   "Number of Redis sentinel nodes (master/slave deployments).",
				PlanModifiers: reqReplaceInt(),
			},
			"sentinel_cloud_profile_names": schema.ListAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Cloud profile names for Redis sentinels when sentinel_count exceeds server_count.",
				PlanModifiers: []planmodifier.List{listRequiresReplace()},
			},
			"encrypt_disk": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Encrypt the data disk.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},
			"enable_ssl": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable SSL/TLS for client connections.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},
			"paused": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether the cluster is paused. Toggling this pauses or resumes the cluster.",
			},

			// MongoDB-specific
			"mongo_engine": schema.StringAttribute{
				Optional:      true,
				Description:   "MongoDB storage engine: `wiredtiger` (default) or `mmapv1`.",
				PlanModifiers: reqReplaceStr(),
			},
			"compression_algo": schema.StringAttribute{
				Optional:      true,
				Description:   "MongoDB compression algorithm: `snappy`, `zlib`, or `zstd`.",
				PlanModifiers: reqReplaceStr(),
			},

			// Redis-specific
			"cluster_mode": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable Redis cluster mode.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},
			"backup_interval_hours": schema.Int64Attribute{
				Optional:      true,
				Description:   "Redis scheduled backup interval in hours (0 disables).",
				PlanModifiers: reqReplaceInt(),
			},
			"maxmemory_policy": schema.StringAttribute{
				Optional:      true,
				Description:   "Redis eviction policy (e.g. `noeviction`, `allkeys-lru`).",
				PlanModifiers: reqReplaceStr(),
			},
			"enable_rdb": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable Redis RDB snapshots.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},
			"enable_aof": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable Redis AOF persistence.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},

			// MySQL-specific
			"replica_config": schema.Int64Attribute{
				Optional:      true,
				Description:   "MySQL replication: 0 standalone, 1 semisync, 2 async.",
				PlanModifiers: reqReplaceInt(),
			},

			// PostgreSQL-specific
			"replication_type": schema.StringAttribute{
				Optional:      true,
				Description:   "PostgreSQL replication type: `ASYNC` or `SYNC`.",
				PlanModifiers: reqReplaceStr(),
			},
			"sync_commit_type": schema.StringAttribute{
				Optional:      true,
				Description:   "PostgreSQL synchronous commit type (e.g. `LOCAL`, `ON`, `REMOTE_WRITE`).",
				PlanModifiers: reqReplaceStr(),
			},
			"enable_pgbouncer": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				Description:   "Enable PgBouncer connection pooling.",
				PlanModifiers: []planmodifier.Bool{boolRequiresReplace()},
			},

			// Computed
			"status":             schema.StringAttribute{Computed: true, Description: "Current lifecycle status."},
			"cluster_type":       schema.StringAttribute{Computed: true, Description: "Topology (Standalone, ReplicaSet, Sharded)."},
			"disk_size_gb":       schema.Int64Attribute{Computed: true, Description: "Provisioned disk size in GB."},
			"encryption_enabled": schema.BoolAttribute{Computed: true, Description: "Whether encryption at rest is active."},
			"ssl_active":         schema.BoolAttribute{Computed: true, Description: "Whether SSL is active on the cluster."},
		},
	}
}

func (r *clusterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	c, err := clientFromProviderData(req.ProviderData)
	if err != nil {
		resp.Diagnostics.AddError("Unexpected provider data", err.Error())
		return
	}
	r.client = c
}

func (r *clusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	profileNames, d := stringsFromList(ctx, plan.CloudProfileNames)
	resp.Diagnostics.Append(d...)
	sentinelNames, d := stringsFromList(ctx, plan.SentinelProfiles)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	poolIDs, err := r.resolveProfiles(ctx, profileNames)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving cloud profiles", err.Error())
		return
	}
	sentinelIDs, err := r.resolveProfiles(ctx, sentinelNames)
	if err != nil {
		resp.Diagnostics.AddError("Error resolving sentinel cloud profiles", err.Error())
		return
	}

	in := client.CreateClusterInput{
		DBType:                db,
		Name:                  plan.Name.ValueString(),
		Size:                  plan.Size.ValueString(),
		Version:               plan.Version.ValueString(),
		ShardCount:            int(plan.ShardCount.ValueInt64()),
		ReplicaCount:          int(plan.ReplicaCount.ValueInt64()),
		ServerCount:           int(plan.ServerCount.ValueInt64()),
		SentinelCount:         int(plan.SentinelCount.ValueInt64()),
		MachinePoolIDs:        poolIDs,
		SentinelPools:         sentinelIDs,
		EncryptDisk:           plan.EncryptDisk.ValueBool(),
		EnableSSL:             plan.EnableSSL.ValueBool(),
		MongoEngine:           stringValue(plan.MongoEngine),
		CompressionAlgo:       stringValue(plan.CompressionAlgo),
		ClusterMode:           plan.ClusterMode.ValueBool(),
		BackupIntervalInHours: int(plan.BackupIntervalHours.ValueInt64()),
		MaxMemoryPolicy:       stringValue(plan.MaxMemoryPolicy),
		EnableRDB:             plan.EnableRDB.ValueBool(),
		EnableAOF:             plan.EnableAOF.ValueBool(),
		ReplicaConfig:         int(plan.ReplicaConfig.ValueInt64()),
		ReplicationType:       stringValue(plan.ReplicationType),
		SyncCommitType:        stringValue(plan.SyncCommitType),
		EnablePgBouncer:       plan.EnablePgBouncer.ValueBool(),
	}

	clusterID, actionID, err := r.client.CreateCluster(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating cluster", err.Error())
		return
	}
	plan.ID = types.StringValue(clusterID)
	// Persist the ID immediately so a failed wait still leaves a deletable resource.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), clusterID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("database"), plan.Database)...)

	tflog.Info(ctx, "waiting for cluster provisioning", map[string]any{"cluster_id": clusterID, "action_id": actionID})
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for cluster provisioning", err.Error())
		return
	}

	if plan.Paused.ValueBool() {
		if _, err := r.client.PauseCluster(ctx, db, clusterID); err != nil {
			resp.Diagnostics.AddError("Error pausing cluster after creation", err.Error())
			return
		}
	}

	cluster, err := r.client.GetCluster(ctx, db, clusterID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after creation", err.Error())
		return
	}
	r.mapComputed(cluster, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	cluster, err := r.client.GetCluster(ctx, db, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading cluster", err.Error())
		return
	}
	r.mapComputed(cluster, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *clusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(plan.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}
	id := state.ID.ValueString()

	if plan.Size.ValueString() != state.Size.ValueString() {
		actionID, err := r.client.ScaleCluster(ctx, db, id, plan.Size.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error scaling cluster", err.Error())
			return
		}
		if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
			resp.Diagnostics.AddError("Error waiting for scale operation", err.Error())
			return
		}
	}

	if plan.Paused.ValueBool() != state.Paused.ValueBool() {
		var actionID string
		var err error
		if plan.Paused.ValueBool() {
			actionID, err = r.client.PauseCluster(ctx, db, id)
		} else {
			actionID, err = r.client.ResumeCluster(ctx, db, id)
		}
		if err != nil {
			resp.Diagnostics.AddError("Error changing cluster power state", err.Error())
			return
		}
		if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
			resp.Diagnostics.AddError("Error waiting for pause/resume", err.Error())
			return
		}
	}

	cluster, err := r.client.GetCluster(ctx, db, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading cluster after update", err.Error())
		return
	}
	plan.ID = state.ID
	r.mapComputed(cluster, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *clusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, ok := parseDBTypeDiag(state.Database.ValueString(), &resp.Diagnostics)
	if !ok {
		return
	}

	actionID, err := r.client.DeleteCluster(ctx, db, state.ID.ValueString(), true)
	if err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting cluster", err.Error())
		return
	}
	if err := r.client.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
		resp.Diagnostics.AddError("Error waiting for cluster deletion", err.Error())
	}
}

// ImportState accepts "<database>:<cluster_id>".
func (r *clusterResource) resolveProfiles(ctx context.Context, names []string) ([]string, error) {
	ids := make([]string, 0, len(names))
	for _, name := range names {
		profile, err := r.client.FindCloudProfileByName(ctx, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, profile.ID)
	}
	return ids, nil
}

func (r *clusterResource) mapComputed(cluster *client.Cluster, model *clusterResourceModel) {
	model.ID = types.StringValue(cluster.ID)
	model.Status = optionalString(cluster.Status)
	model.ClusterType = optionalString(cluster.ClusterType)
	model.DiskSizeGB = types.Int64Value(cluster.DiskSizeGB)
	model.EncryptionEnabled = types.BoolValue(cluster.EncryptionEnabled)
	model.SSLActive = types.BoolValue(cluster.SSLEnabled)
	if cluster.Size != "" {
		if norm, ok := client.NormalizeSize(cluster.Size); ok {
			model.Size = types.StringValue(norm)
		}
	}
}
