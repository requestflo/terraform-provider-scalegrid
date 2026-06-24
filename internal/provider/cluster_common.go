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

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

// clusterPollInterval controls how often asynchronous cluster jobs are polled.
const clusterPollInterval = 20 * time.Second

func reqReplaceStr() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

func reqReplaceInt() []planmodifier.Int64 {
	return []planmodifier.Int64{int64planmodifier.RequiresReplace()}
}

// commonClusterAttributes returns the schema attributes shared by every
// per-engine cluster resource (scalegrid_mongodb_cluster, _redis_cluster, etc.).
// Each engine resource merges these with its engine-specific attributes.
func commonClusterAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "Unique identifier of the cluster.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
			Description: "Names of the ScaleGrid cloud profiles to deploy nodes into (one per node across " +
				"shards). Required for all plans: a shared profile for Dedicated (ScaleGrid-hosted) plans, " +
				"or your own profile for Bring Your Own Cloud. Look them up with the `scalegrid_cloud_profile` data source.",
			PlanModifiers: []planmodifier.List{listRequiresReplace()},
		},
		"shard_count": schema.Int64Attribute{
			Optional:      true,
			Computed:      true,
			Default:       int64default.StaticInt64(1),
			Description:   "Number of shards. 1 for standalone/replica set; more for sharded.",
			PlanModifiers: reqReplaceInt(),
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
			Description: "Whether the cluster is paused. Toggling this pauses or resumes the cluster in place. " +
				"Note: pause/resume is only supported for Bring Your Own Cloud (BYOC) deployments.",
		},

		// Computed / read-only.
		"status":             schema.StringAttribute{Computed: true, Description: "Current lifecycle status."},
		"cluster_type":       schema.StringAttribute{Computed: true, Description: "Topology (Standalone, ReplicaSet, Sharded)."},
		"disk_size_gb":       schema.Int64Attribute{Computed: true, Description: "Provisioned disk size in GB."},
		"encryption_enabled": schema.BoolAttribute{Computed: true, Description: "Whether encryption at rest is active."},
		"ssl_active":         schema.BoolAttribute{Computed: true, Description: "Whether SSL is active on the cluster."},
	}
}

// mergeAttributes combines the common attributes with engine-specific ones.
func mergeAttributes(extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := commonClusterAttributes()
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// resolveProfiles maps cloud profile names to machine pool IDs.
func resolveProfiles(ctx context.Context, c *client.Client, names []string) ([]string, error) {
	ids := make([]string, 0, len(names))
	for _, name := range names {
		profile, err := c.FindCloudProfileByName(ctx, name)
		if err != nil {
			return nil, err
		}
		ids = append(ids, profile.ID)
	}
	return ids, nil
}

// clusterComputed holds the read-only values mapped back from the API.
type clusterComputed struct {
	Size              types.String
	Status            types.String
	ClusterType       types.String
	DiskSizeGB        types.Int64
	EncryptionEnabled types.Bool
	SSLActive         types.Bool
}

// computedFromCluster extracts the shared computed values from an API cluster.
func computedFromCluster(cluster *client.Cluster) clusterComputed {
	cc := clusterComputed{
		Status:            optionalString(cluster.Status),
		ClusterType:       optionalString(cluster.ClusterType),
		DiskSizeGB:        types.Int64Value(cluster.DiskSizeGB),
		EncryptionEnabled: types.BoolValue(cluster.EncryptionEnabled),
		SSLActive:         types.BoolValue(cluster.SSLEnabled),
	}
	if cluster.Size != "" {
		if norm, ok := client.NormalizeSize(cluster.Size); ok {
			cc.Size = types.StringValue(norm)
		}
	}
	return cc
}

// provisionResult carries the data a resource needs after provisioning.
type provisionResult struct {
	ClusterID string
	Cluster   *client.Cluster
}

// scaleAndPause applies the two in-place mutations shared by every engine:
// resizing and pausing/resuming. It waits for each job to finish.
func scaleAndPause(ctx context.Context, c *client.Client, db client.DBType, id string,
	planSize, stateSize string, planPaused, statePaused bool) error {
	if planSize != stateSize {
		actionID, err := c.ScaleCluster(ctx, db, id, planSize)
		if err != nil {
			return err
		}
		if err := c.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
			return err
		}
	}
	if planPaused != statePaused {
		var actionID string
		var err error
		if planPaused {
			actionID, err = c.PauseCluster(ctx, db, id)
		} else {
			actionID, err = c.ResumeCluster(ctx, db, id)
		}
		if err != nil {
			return err
		}
		if err := c.WaitForAction(ctx, actionID, clusterPollInterval); err != nil {
			return err
		}
	}
	return nil
}

// deleteCluster tears down a cluster and waits for completion, treating an
// already-absent cluster as success.
func deleteCluster(ctx context.Context, c *client.Client, db client.DBType, id string) error {
	actionID, err := c.DeleteCluster(ctx, db, id, true)
	if err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	return c.WaitForAction(ctx, actionID, clusterPollInterval)
}

// persistIDEarly writes the cluster id (and engine, where applicable) to state
// before the provisioning wait, so a timeout still leaves a deletable resource.
func persistIDEarly(ctx context.Context, state *resource.CreateResponse, id string) {
	state.Diagnostics.Append(state.State.SetAttribute(ctx, path.Root("id"), id)...)
}
