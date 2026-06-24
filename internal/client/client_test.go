package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// envelope writes a ScaleGrid-style success envelope merged with extra fields.
func writeEnvelope(w http.ResponseWriter, extra map[string]any) {
	body := map[string]any{"error": map[string]any{"code": "Success"}}
	for k, v := range extra {
		body[k] = v
	}
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code, msg string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"code": code, "errorMessageWithDetails": msg},
	})
}

// newTestClient returns a client pointed at srv without performing login.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(context.Background(), Config{BaseURL: srv.URL, SkipLogin: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c
}

func TestLoginSuccess(t *testing.T) {
	var gotBody map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc"})
		writeEnvelope(w, nil)
	}))
	defer srv.Close()

	_, err := NewClient(context.Background(), Config{BaseURL: srv.URL, Email: "a@b.com", Password: "pw"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if gotBody["username"] != "a@b.com" || gotBody["password"] != "pw" {
		t.Errorf("unexpected login body: %+v", gotBody)
	}
}

func TestLoginTwoFactorRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, "TwoFactorAuthNeeded", "")
	}))
	defer srv.Close()
	_, err := NewClient(context.Background(), Config{BaseURL: srv.URL, Email: "a@b.com", Password: "pw"})
	if err == nil {
		t.Fatal("expected 2FA error")
	}
}

func TestErrorEnvelopeAndNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, "ClusterNotFound", "The cluster was not found")
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	_, err := c.GetCluster(context.Background(), DBMongo, "missing")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound, got: %v", err)
	}
}

func TestCreateClusterMongo(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		// The API returns IDs as JSON integers.
		writeEnvelope(w, map[string]any{"clusterID": 123, "actionID": 1})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	id, action, err := c.CreateCluster(context.Background(), CreateClusterInput{
		DBType: DBMongo, Name: "prod", Size: "Small", Version: "7.0",
		ShardCount: 1, ReplicaCount: 3, MachinePoolIDs: []string{"m1", "m2", "m3"},
	})
	if err != nil {
		t.Fatalf("CreateCluster: %v", err)
	}
	if id != "123" || action != "1" {
		t.Errorf("unexpected ids: %q %q", id, action)
	}
	if gotPath != "/MongoClusters/create" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if gotBody["clusterName"] != "prod" || gotBody["version"] != "7.0" {
		t.Errorf("unexpected body: %+v", gotBody)
	}
	// enableAuth is not part of the documented API and must not be sent.
	if _, ok := gotBody["enableAuth"]; ok {
		t.Errorf("enableAuth should not be present: %+v", gotBody)
	}
	if gotBody["engine"] != "wiredtiger" {
		t.Errorf("expected engine wiredtiger: %+v", gotBody)
	}
}

// TestListClustersKeyAndIntegerIDs verifies the singular "cluster" list key and
// integer-ID decoding used by Mongo/Redis/MySQL, and the lowercase PostgreSQL
// list path with its "clusters" key.
func TestListClustersKeyAndIntegerIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/MongoClusters/list":
			writeEnvelope(w, map[string]any{"cluster": []map[string]any{
				{"id": 42, "name": "m"},
			}})
		case "/postgresqlclusters/list":
			writeEnvelope(w, map[string]any{"clusters": []map[string]any{
				{"id": 7, "name": "p"},
			}})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	mongo, err := c.ListClusters(context.Background(), DBMongo)
	if err != nil {
		t.Fatalf("ListClusters mongo: %v", err)
	}
	if len(mongo) != 1 || mongo[0].ID != "42" || mongo[0].Name != "m" {
		t.Errorf("unexpected mongo clusters: %+v", mongo)
	}

	pg, err := c.ListClusters(context.Background(), DBPostgreSQL)
	if err != nil {
		t.Fatalf("ListClusters pg: %v", err)
	}
	if len(pg) != 1 || pg[0].ID != "7" {
		t.Errorf("unexpected pg clusters: %+v", pg)
	}
}

func TestCreateClusterRequiresMachinePool(t *testing.T) {
	c := newTestClient(t, httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})))
	_, _, err := c.CreateCluster(context.Background(), CreateClusterInput{DBType: DBRedis, Name: "x"})
	if err == nil {
		t.Fatal("expected error when no machine pools provided")
	}
}

func TestWaitForAction(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		status := ActionRunning
		if calls >= 2 {
			status = ActionCompleted
		}
		writeEnvelope(w, map[string]any{"action": map[string]any{"id": "a-1", "status": status}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if err := c.WaitForAction(context.Background(), "a-1", 5*time.Millisecond); err != nil {
		t.Fatalf("WaitForAction: %v", err)
	}
	if calls < 2 {
		t.Errorf("expected >=2 polls, got %d", calls)
	}
}

// TestWaitForActionTopLevel verifies the action status is read from the
// top-level response fields (the shape documented in the OpenAPI spec), not only
// from a nested "action" object.
func TestWaitForActionTopLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, map[string]any{"status": ActionCompleted, "progress": 100})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if err := c.WaitForAction(context.Background(), "a-1", time.Millisecond); err != nil {
		t.Fatalf("WaitForAction: %v", err)
	}
}

// TestWaitForActionFailedTopLevel verifies a failed job surfaces the message
// from the top-level "error" object.
func TestWaitForActionFailedTopLevel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": ActionFailed,
			"error":  map[string]any{"code": "Success", "errorMessage": "disk full"},
		})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	err := c.WaitForAction(context.Background(), "a-1", time.Millisecond)
	if err == nil {
		t.Fatal("expected failure error")
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Errorf("expected failure message to include API detail, got: %v", err)
	}
}

func TestWaitForActionFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeEnvelope(w, map[string]any{"action": map[string]any{
			"id": "a-1", "status": ActionFailed,
			"stepError": map[string]any{"errorMessageWithDetails": "boom"},
		}})
	}))
	defer srv.Close()
	c := newTestClient(t, srv)
	if err := c.WaitForAction(context.Background(), "a-1", time.Millisecond); err == nil {
		t.Fatal("expected failure error")
	}
}

func TestFirewallRoundTrip(t *testing.T) {
	var setCalls, configureCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Clusters/setClusterLevelIPWhiteList":
			setCalls++
			writeEnvelope(w, nil)
		case "/Clusters/configureIPWhiteList":
			configureCalls++
			writeEnvelope(w, nil)
		case "/Clusters/getClusterLevelIPWhiteList":
			writeEnvelope(w, map[string]any{"cidrList": []string{"10.0.0.0/8"}})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	c := newTestClient(t, srv)

	if err := c.SetFirewallRules(context.Background(), DBPostgreSQL, "c1", []string{"10.0.0.0/8"}); err != nil {
		t.Fatalf("SetFirewallRules: %v", err)
	}
	if setCalls != 1 || configureCalls != 1 {
		t.Errorf("expected one call each, got set=%d configure=%d", setCalls, configureCalls)
	}
	cidrs, err := c.GetFirewallRules(context.Background(), DBPostgreSQL, "c1")
	if err != nil {
		t.Fatalf("GetFirewallRules: %v", err)
	}
	if len(cidrs) != 1 || cidrs[0] != "10.0.0.0/8" {
		t.Errorf("unexpected cidrs: %+v", cidrs)
	}
}

func TestDBTypeHelpers(t *testing.T) {
	if DBMongo.PathPrefix() != "MongoClusters" {
		t.Errorf("path prefix: %s", DBMongo.PathPrefix())
	}
	if DBPostgreSQL.listPrefix() != "postgresqlclusters" {
		t.Errorf("pg list prefix: %s", DBPostgreSQL.listPrefix())
	}
	if DBMongo.listPrefix() != "MongoClusters" {
		t.Errorf("mongo list prefix: %s", DBMongo.listPrefix())
	}
	if DBMongo.WireType() != "MONGODB" {
		t.Errorf("wire type: %s", DBMongo.WireType())
	}
	if DBPostgreSQL.WireType() != "POSTGRESQL" {
		t.Errorf("wire type: %s", DBPostgreSQL.WireType())
	}
	if db, ok := ParseDBType("postgres"); !ok || db != DBPostgreSQL {
		t.Errorf("parse postgres: %v %v", db, ok)
	}
	if _, ok := ParseDBType("oracle"); ok {
		t.Error("expected oracle to be invalid")
	}
	if s, ok := NormalizeSize("x2xlarge"); !ok || s != "X2XLarge" {
		t.Errorf("normalize size: %q %v", s, ok)
	}
}
