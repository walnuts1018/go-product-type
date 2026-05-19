package multi

import "testing"

func TestGeneratedOutputsBuildAndBehave(t *testing.T) {
	alpha := Alpha(&Left{Name: "left"})
	left, ok := alpha.AsLeft()
	if !ok {
		t.Fatal("Alpha.AsLeft() ok = false, want true")
	}
	if left.Name != "left" {
		t.Fatalf("Alpha.AsLeft().Name = %q, want %q", left.Name, "left")
	}
	if _, ok := alpha.AsRight(); ok {
		t.Fatal("Alpha.AsRight() ok = true, want false")
	}
	got := MatchAlpha(alpha,
		func(v Left) string { return v.Name },
		func(v Right) string { return "unexpected" },
	)
	if got != "left" {
		t.Fatalf("MatchAlpha() = %q, want %q", got, "left")
	}

	beta := NewBeta(Primary{ID: "id-1"}, Secondary{Enabled: true})
	if beta.ID != "id-1" {
		t.Fatalf("NewBeta().ID = %q, want %q", beta.ID, "id-1")
	}
	if !beta.Enabled {
		t.Fatal("NewBeta().Enabled = false, want true")
	}
	if primary := beta.ToPrimary(); primary.ID != "id-1" {
		t.Fatalf("Beta.ToPrimary().ID = %q, want %q", primary.ID, "id-1")
	}
	if secondary := beta.ToSecondary(); !secondary.Enabled {
		t.Fatal("Beta.ToSecondary().Enabled = false, want true")
	}
}
