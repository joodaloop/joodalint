package rules

import "testing"

func TestDuplicateIDs_Flagged(t *testing.T) {
	f := &HTMLFile{Path: "test.html", IDs: map[string]int{"intro": 2, "outro": 1}}
	diags := duplicateIDs{}.Check(f, nil)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag, got %v", messages(diags))
	}
	want := `id "intro" appears 2 times`
	if diags[0].Message != want {
		t.Errorf("message = %q, want %q", diags[0].Message, want)
	}
}

func TestDuplicateIDs_Deterministic(t *testing.T) {
	f := &HTMLFile{Path: "test.html", IDs: map[string]int{"zeta": 3, "alpha": 2}}
	diags := duplicateIDs{}.Check(f, nil)
	if len(diags) != 2 {
		t.Fatalf("want 2 diags, got %v", messages(diags))
	}
	if diags[0].Message > diags[1].Message {
		t.Errorf("diags not sorted: %v", messages(diags))
	}
}

func TestDuplicateIDs_NoneUnique(t *testing.T) {
	f := &HTMLFile{Path: "test.html", IDs: map[string]int{"a": 1, "b": 1}}
	assertNoDiags(t, duplicateIDs{}.Check(f, nil))
}

func TestDuplicateIDs_ID(t *testing.T) {
	if (duplicateIDs{}).ID() != "duplicate-id" {
		t.Fatal("wrong ID")
	}
}
