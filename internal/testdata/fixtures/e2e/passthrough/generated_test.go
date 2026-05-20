package passthrough

import "testing"

func TestPassthroughGeneratedFileExposesInlineCode(t *testing.T) {
	if got := FormatInline(DefaultInline); got != "inline:inline" {
		t.Fatalf("got %q, want %q", got, "inline:inline")
	}

	combined := NewCombined(DefaultInline, Shared{Count: 2})
	if combined.Label != "inline" {
		t.Fatalf("got label %q, want %q", combined.Label, "inline")
	}
	if combined.Count != 2 {
		t.Fatalf("got count %d, want %d", combined.Count, 2)
	}
}
