package rules

import "testing"

func TestFormattingShortSpansOK(t *testing.T) {
	src := "Some *short italic*, **bold text**, and `code` here.\n"
	diags := markdownFormatting{}.Check(mdFile(src), nil)
	assertNoDiags(t, diags)
}

func TestFormattingLongItalic(t *testing.T) {
	// 101-character italic span — should warn.
	text := "*" + repeat("a", 101) + "*\n"
	diags := markdownFormatting{}.Check(mdFile(text), nil)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag, got %d: %v", len(diags), messages(diags))
	}
	if !containsMsg(diags, "emphasis span 101 chars") {
		t.Fatalf("want emphasis warning, got %v", messages(diags))
	}
	if diags[0].Line != 1 {
		t.Fatalf("want line 1, got %d", diags[0].Line)
	}
}

func TestFormattingLongBold(t *testing.T) {
	text := "**" + repeat("b", 120) + "**\n"
	diags := markdownFormatting{}.Check(mdFile(text), nil)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag, got %d: %v", len(diags), messages(diags))
	}
	if !containsMsg(diags, "**bold**") {
		t.Fatalf("want bold warning, got %v", messages(diags))
	}
}

func TestFormattingLongCode(t *testing.T) {
	text := "`" + repeat("c", 150) + "`\n"
	diags := markdownFormatting{}.Check(mdFile(text), nil)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag, got %d: %v", len(diags), messages(diags))
	}
	if !containsMsg(diags, "inline code span 150 chars") {
		t.Fatalf("want code warning, got %v", messages(diags))
	}
}

func TestFormattingCodeAtBoundary(t *testing.T) {
	// Exactly 100 characters — should NOT warn.
	text := "`" + repeat("x", 100) + "`\n"
	diags := markdownFormatting{}.Check(mdFile(text), nil)
	assertNoDiags(t, diags)
}

func TestFormattingMultiLineSpan(t *testing.T) {
	// Emphasis wrapping across lines (no blank line) with >100 chars.
	text := "*this wraps\nacross two lines " + repeat("x", 100) + " and ends here*\n"
	diags := markdownFormatting{}.Check(mdFile(text), nil)
	if len(diags) != 1 {
		t.Fatalf("want 1 diag for multi-line emphasis, got %d: %v", len(diags), messages(diags))
	}
	if !containsMsg(diags, "emphasis span") {
		t.Fatalf("want emphasis warning, got %v", messages(diags))
	}
}

func TestFormattingID(t *testing.T) {
	if (markdownFormatting{}).ID() != "formatting" {
		t.Fatal("wrong ID")
	}
}

// repeat returns s repeated n times.
func repeat(s string, n int) string {
	out := make([]byte, 0, n*len(s))
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
