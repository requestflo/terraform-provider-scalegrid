package client

import (
	"encoding/json"
	"strings"
)

// rawID converts a JSON value that may be a number or a quoted string into a
// plain string. The ScaleGrid API returns most IDs as integers, but the
// provider tracks them as strings.
func rawID(raw json.RawMessage) string {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return ""
	}
	s = strings.TrimPrefix(s, `"`)
	s = strings.TrimSuffix(s, `"`)
	return s
}

// DBType identifies a database engine. The string value is the path prefix used
// by the console API (e.g. "Mongo" -> /MongoClusters/...). The wire value sent
// in request bodies (the "dbType" field) is the upper-cased canonical name.
type DBType string

const (
	DBMongo      DBType = "Mongo"
	DBRedis      DBType = "Redis"
	DBMySQL      DBType = "MySQL"
	DBPostgreSQL DBType = "PostgreSQL"
)

// PathPrefix returns the canonical cluster endpoint prefix, e.g. "MongoClusters".
// PostgreSQL uses this capitalized form for a handful of endpoints (create,
// deletebackup, getBackupSchedule, enablePgBouncer).
func (d DBType) PathPrefix() string { return string(d) + "Clusters" }

// listPrefix returns the prefix used by the bulk of per-cluster operations
// (list, fetch, scale, delete, getCredentials, backup, restore). PostgreSQL
// serves these from an all-lowercase path, unlike the other engines.
func (d DBType) listPrefix() string {
	if d == DBPostgreSQL {
		return "postgresqlclusters"
	}
	return string(d) + "Clusters"
}

// WireType returns the upper-cased dbType value used in generic request bodies
// (MONGODB, REDIS, MYSQL, POSTGRESQL).
func (d DBType) WireType() string {
	if d == DBMongo {
		return "MONGODB"
	}
	return strings.ToUpper(string(d))
}

// ParseDBType maps a user-supplied database name to a DBType.
func ParseDBType(s string) (DBType, bool) {
	switch strings.ToLower(s) {
	case "mongo", "mongodb":
		return DBMongo, true
	case "redis":
		return DBRedis, true
	case "mysql":
		return DBMySQL, true
	case "postgresql", "postgres":
		return DBPostgreSQL, true
	default:
		return "", false
	}
}

// Valid cluster sizes (t-shirt tiers).
var ValidSizes = []string{"Micro", "Small", "Medium", "Large", "XLarge", "X2XLarge", "X4XLarge"}

// NormalizeSize canonicalises a user-supplied size to ScaleGrid's casing.
func NormalizeSize(s string) (string, bool) {
	switch strings.ToLower(s) {
	case "micro":
		return "Micro", true
	case "small":
		return "Small", true
	case "medium":
		return "Medium", true
	case "large":
		return "Large", true
	case "xlarge":
		return "XLarge", true
	case "x2xlarge":
		return "X2XLarge", true
	case "x4xlarge":
		return "X4XLarge", true
	default:
		return "", false
	}
}

// Cluster is the subset of cluster fields the provider tracks. The console
// returns many more fields; only those used by Terraform are mapped here.
type Cluster struct {
	ID                string             `json:"id,omitempty"`
	Name              string             `json:"name,omitempty"`
	ClusterType       string             `json:"clusterType,omitempty"`
	Status            string             `json:"status,omitempty"`
	Size              string             `json:"size,omitempty"`
	VersionStr        string             `json:"versionStr,omitempty"`
	DiskSizeGB        int64              `json:"diskSizeGB,omitempty"`
	UsedDiskSizeGB    float64            `json:"usedDiskSizeGB,omitempty"`
	RAMGB             float64            `json:"ramGB,omitempty"`
	SSLEnabled        bool               `json:"sslEnabled,omitempty"`
	EncryptionEnabled bool               `json:"encryptionEnabled,omitempty"`
	Engine            string             `json:"engine,omitempty"`
	CompressionAlgo   string             `json:"compressionAlgo,omitempty"`
	ClusterMode       bool               `json:"clusterMode,omitempty"`
	ConnectionString  []ConnectionString `json:"connectionString,omitempty"`
	CommandLineString string             `json:"commandLineString,omitempty"`
	CommandLineServer string             `json:"commandLineServer,omitempty"`
}

// UnmarshalJSON decodes a Cluster, tolerating the API's quirks: numeric IDs
// (returned as JSON integers) and a connectionString field that is an array on
// some endpoints and an object on others.
func (c *Cluster) UnmarshalJSON(data []byte) error {
	type alias Cluster
	aux := struct {
		ID               json.RawMessage `json:"id"`
		ConnectionString json.RawMessage `json:"connectionString"`
		*alias
	}{alias: (*alias)(c)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	c.ID = rawID(aux.ID)
	c.ConnectionString = nil
	if len(aux.ConnectionString) > 0 && aux.ConnectionString[0] == '[' {
		// Only the array form carries the driver-specific connection strings we
		// surface; the object form (used by some fetch endpoints) is ignored.
		_ = json.Unmarshal(aux.ConnectionString, &c.ConnectionString)
	}
	return nil
}

// ConnectionString is one driver-specific connection string for a cluster.
type ConnectionString struct {
	Driver  string `json:"driver,omitempty"`
	ConnStr string `json:"conString,omitempty"`
}

// clusterListResponse wraps GET /{Db}Clusters/list. Mongo, Redis and MySQL
// return the array under "cluster" (singular); PostgreSQL uses "clusters".
type clusterListResponse struct {
	Cluster  []Cluster `json:"cluster"`
	Clusters []Cluster `json:"clusters"`
}

// clusters returns whichever key the engine populated.
func (r clusterListResponse) clusters() []Cluster {
	if len(r.Clusters) > 0 {
		return r.Clusters
	}
	return r.Cluster
}

// clusterFetchResponse wraps GET /{Db}Clusters/{id}/fetch.
type clusterFetchResponse struct {
	Cluster Cluster `json:"cluster"`
}

// asyncResponse carries the IDs returned by mutating endpoints. The API returns
// these as integers, and is inconsistent about casing (actionID vs actionId,
// machinePoolID vs machinePoolId), so we decode every spelling.
type asyncResponse struct {
	ClusterID     string
	MachinePoolID string
	ActionID      string
}

func (a *asyncResponse) UnmarshalJSON(data []byte) error {
	var aux struct {
		ClusterIDUpper     json.RawMessage `json:"clusterID"`
		ClusterIDLower     json.RawMessage `json:"clusterId"`
		MachinePoolIDUpper json.RawMessage `json:"machinePoolID"`
		MachinePoolIDLower json.RawMessage `json:"machinePoolId"`
		ActionIDUpper      json.RawMessage `json:"actionID"`
		ActionIDLower      json.RawMessage `json:"actionId"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	a.ClusterID = firstRawID(aux.ClusterIDUpper, aux.ClusterIDLower)
	a.MachinePoolID = firstRawID(aux.MachinePoolIDUpper, aux.MachinePoolIDLower)
	a.ActionID = firstRawID(aux.ActionIDUpper, aux.ActionIDLower)
	return nil
}

// firstRawID returns the first non-empty decoded ID from the candidates.
func firstRawID(raws ...json.RawMessage) string {
	for _, raw := range raws {
		if id := rawID(raw); id != "" {
			return id
		}
	}
	return ""
}

// CreateClusterInput is the engine-agnostic input the provider supplies. The
// client translates it into the per-engine request body.
type CreateClusterInput struct {
	DBType         DBType
	Name           string
	Size           string
	Version        string
	ShardCount     int
	ReplicaCount   int // mongo/mysql/postgres nodes per shard
	ServerCount    int // redis nodes per shard
	SentinelCount  int // redis
	MachinePoolIDs []string
	EncryptDisk    bool
	EnableSSL      bool
	CIDRList       []string

	// MongoDB
	MongoEngine     string
	CompressionAlgo string

	// Redis
	ClusterMode           bool
	BackupIntervalInHours int

	// MySQL
	ReplicaConfig int

	// PostgreSQL
	ReplicationType string
	SyncCommitType  string
	EnablePgBouncer bool
}

// CloudProfile is a ScaleGrid cloud profile ("machine pool").
type CloudProfile struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"providerMachinePoolName,omitempty"`
	DBType     string `json:"dbType,omitempty"`
	Type       string `json:"type,omitempty"` // EC2, AZUREARM, DIGITALOCEAN
	Status     string `json:"status,omitempty"`
	Shared     bool   `json:"shared,omitempty"`
	ConfigJSON string `json:"configJSON,omitempty"`
}

// UnmarshalJSON decodes a CloudProfile, converting the numeric id to a string.
func (p *CloudProfile) UnmarshalJSON(data []byte) error {
	type alias CloudProfile
	aux := struct {
		ID json.RawMessage `json:"id"`
		*alias
	}{alias: (*alias)(p)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.ID = rawID(aux.ID)
	return nil
}

// CloudType returns a friendly cloud name (AWS for EC2).
func (p CloudProfile) CloudType() string {
	if p.Type == "EC2" {
		return "AWS"
	}
	return p.Type
}

type cloudProfileListResponse struct {
	Clouds []CloudProfile `json:"clouds"`
}

// CreateAWSCloudProfileInput holds the fields for an AWS (EC2/VPC) cloud profile.
type CreateAWSCloudProfileInput struct {
	DBType             DBType
	Name               string
	Region             string
	AccessKey          string
	SecretKey          string
	VPCID              string
	SubnetID           string
	VPCCIDR            string
	SubnetCIDR         string
	SecurityGroupID    string
	SecurityGroupName  string
	ConnectivityConfig string
	EnableSSH          bool
}

// Backup represents a cluster backup.
type Backup struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	ObjectID string `json:"object_id,omitempty"`
	Created  int64  `json:"created,omitempty"`
	Type     string `json:"type,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// UnmarshalJSON decodes a Backup, converting the numeric id and object_id to
// strings.
func (b *Backup) UnmarshalJSON(data []byte) error {
	type alias Backup
	aux := struct {
		ID       json.RawMessage `json:"id"`
		ObjectID json.RawMessage `json:"object_id"`
		*alias
	}{alias: (*alias)(b)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	b.ID = rawID(aux.ID)
	b.ObjectID = rawID(aux.ObjectID)
	return nil
}

type backupListResponse struct {
	Backups []Backup `json:"backups"`
}

// Credentials are the root database credentials and connection strings.
type Credentials struct {
	User              string
	Password          string
	ConnectionStrings []ConnectionString
	CommandLine       string
}

type credentialsResponse struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

// AlertRule represents a cluster alert rule.
type AlertRule struct {
	ID            string   `json:"id,omitempty"`
	ClusterID     string   `json:"clusterId,omitempty"`
	DatabaseType  string   `json:"databaseType,omitempty"`
	Type          string   `json:"type,omitempty"`
	Metric        string   `json:"metric,omitempty"`
	Operator      string   `json:"operator,omitempty"`
	Threshold     string   `json:"threshold,omitempty"`
	AverageType   string   `json:"averageType,omitempty"`
	Notifications []string `json:"-"`
	Enabled       bool     `json:"enabled,omitempty"`
	Description   string   `json:"alertRuleDescription,omitempty"`
}

type alertRuleCreateResponse struct {
	Rule AlertRule `json:"rule"`
}

// Action is the status of an asynchronous job.
type Action struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	Progress  int64  `json:"progress,omitempty"`
	Cancelled bool   `json:"cancelled,omitempty"`
	StepError struct {
		ErrorMessageWithDetails string `json:"errorMessageWithDetails"`
		RecommendedAction       string `json:"recommendedAction"`
	} `json:"stepError,omitempty"`
}

type actionResponse struct {
	Action Action `json:"action"`
}

// Action status values, as returned by GET /actions/{id}.
const (
	ActionRunning   = "Running"
	ActionCompleted = "Completed"
	ActionFailed    = "Failed"
)

// firewallResponse wraps the cluster-level IP whitelist.
type firewallResponse struct {
	CIDRList []string `json:"cidrList"`
}

// databaseVersionsResponse wraps GET /Clusters/getDatabaseActiveVersions. The
// API returns a plain array of supported version identifiers.
type databaseVersionsResponse struct {
	Versions []string `json:"versions"`
}
