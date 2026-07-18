package rules

import (
	"fmt"
	"sort"
)

func init() {
	RegisterHTML(&duplicateIDs{})
}

type duplicateIDs struct{}

func (duplicateIDs) ID() string { return "duplicate-id" }

func (duplicateIDs) Check(f *HTMLFile, ctx *HTMLContext) []Diagnostic {
	var dups []string
	for id, count := range f.IDs {
		if count > 1 {
			dups = append(dups, id)
		}
	}
	sort.Strings(dups)
	var diags []Diagnostic
	for _, id := range dups {
		diags = append(diags, Diagnostic{
			Path:    f.Path,
			Rule:    "duplicate-id",
			Message: fmt.Sprintf("id %q appears %d times", id, f.IDs[id]),
		})
	}
	return diags
}
