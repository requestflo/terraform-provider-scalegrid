package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/requestflo/scalegrid-terraform/internal/client"
)

// firstNonEmpty returns the first non-empty string from values.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// stringValue extracts a Go string from a types.String, returning "" for
// null/unknown.
func stringValue(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

// optionalString returns null for empty input, otherwise a known value.
func optionalString(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

// clientFromProviderData performs the type assertion shared by Configure methods.
func clientFromProviderData(providerData any) (*client.Client, error) {
	if providerData == nil {
		return nil, nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		return nil, fmt.Errorf("expected *client.Client, got %T; this is a provider bug", providerData)
	}
	return c, nil
}

// stringsFromList converts a Terraform list into a Go string slice.
func stringsFromList(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	return out, diags
}

// stringsToList converts a Go string slice into a Terraform list.
func stringsToList(ctx context.Context, in []string) (types.List, diag.Diagnostics) {
	return types.ListValueFrom(ctx, types.StringType, in)
}

// parseDBTypeDiag resolves a database name to a client.DBType, adding a
// diagnostic on failure.
func parseDBTypeDiag(database string, diags *diag.Diagnostics) (client.DBType, bool) {
	db, ok := client.ParseDBType(database)
	if !ok {
		diags.AddError("Invalid database type",
			fmt.Sprintf("%q is not a supported database. Use mongodb, redis, mysql, or postgresql.", database))
		return "", false
	}
	return db, true
}
