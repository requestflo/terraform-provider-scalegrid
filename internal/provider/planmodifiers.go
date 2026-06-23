package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// boolRequiresReplace forces replacement when a known boolean value changes,
// while remaining a no-op on create and destroy.
func boolRequiresReplace() planmodifier.Bool { return boolReplace{} }

type boolReplace struct{}

func (boolReplace) Description(context.Context) string {
	return "Changing this value forces a new resource."
}
func (boolReplace) MarkdownDescription(context.Context) string {
	return "Changing this value forces a new resource."
}
func (boolReplace) PlanModifyBool(_ context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.StateValue.Equal(req.PlanValue) {
		resp.RequiresReplace = true
	}
}

// listRequiresReplace forces replacement when a known list value changes.
func listRequiresReplace() planmodifier.List { return listReplace{} }

type listReplace struct{}

func (listReplace) Description(context.Context) string {
	return "Changing this value forces a new resource."
}
func (listReplace) MarkdownDescription(context.Context) string {
	return "Changing this value forces a new resource."
}
func (listReplace) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	if !req.StateValue.Equal(req.PlanValue) {
		resp.RequiresReplace = true
	}
}

// splitImportID splits a "<database>:<id>" import identifier.
func splitImportID(s string) (database, id string, ok bool) {
	parts := splitN(s, 2)
	if parts == nil {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// splitN splits s on ":" into exactly n non-empty parts, or returns nil.
func splitN(s string, n int) []string {
	parts := strings.SplitN(s, ":", n)
	if len(parts) != n {
		return nil
	}
	for _, p := range parts {
		if p == "" {
			return nil
		}
	}
	return parts
}
