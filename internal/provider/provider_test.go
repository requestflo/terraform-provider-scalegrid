package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestProviderSchemaValid(t *testing.T) {
	p := New("test")()
	resp := &provider.SchemaResponse{}
	p.Schema(context.Background(), provider.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("provider schema diagnostics: %v", resp.Diagnostics)
	}
	for _, attr := range []string{"email", "password", "base_url", "two_factor_code"} {
		if _, ok := resp.Schema.Attributes[attr]; !ok {
			t.Errorf("expected provider attribute %q", attr)
		}
	}
	if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("provider schema invalid: %v", diags)
	}
}

func TestResourceSchemasValid(t *testing.T) {
	p := New("test")().(*ScaleGridProvider)
	ctors := p.Resources(context.Background())
	if len(ctors) != 10 {
		t.Errorf("expected 10 resources, got %d", len(ctors))
	}
	for _, ctor := range ctors {
		r := ctor()
		resp := &resource.SchemaResponse{}
		r.Schema(context.Background(), resource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("resource schema diagnostics: %v", resp.Diagnostics)
			continue
		}
		if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
			t.Errorf("resource schema invalid: %v", diags)
		}
	}
}

func TestDataSourceSchemasValid(t *testing.T) {
	p := New("test")().(*ScaleGridProvider)
	ctors := p.DataSources(context.Background())
	if len(ctors) != 5 {
		t.Errorf("expected 5 data sources, got %d", len(ctors))
	}
	for _, ctor := range ctors {
		ds := ctor()
		resp := &datasource.SchemaResponse{}
		ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("data source schema diagnostics: %v", resp.Diagnostics)
			continue
		}
		if diags := resp.Schema.ValidateImplementation(context.Background()); diags.HasError() {
			t.Errorf("data source schema invalid: %v", diags)
		}
	}
}

func TestSplitImportID(t *testing.T) {
	db, id, ok := splitImportID("mongodb:abc123")
	if !ok || db != "mongodb" || id != "abc123" {
		t.Errorf("unexpected: %q %q %v", db, id, ok)
	}
	if _, _, ok := splitImportID("noseparator"); ok {
		t.Error("expected failure for missing separator")
	}
	if parts := splitN("a:b:c", 3); parts == nil || parts[2] != "c" {
		t.Errorf("splitN failed: %v", parts)
	}
}
