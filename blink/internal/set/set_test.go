package set

import "testing"

func TestGet(t *testing.T) {
	if _, ok := Get("spaces"); !ok {
		t.Fatal("spaces set should exist")
	}
	if _, ok := Get("does-not-exist"); ok {
		t.Fatal("unknown set should not resolve")
	}
}

func TestSpacesLevelClamps(t *testing.T) {
	full := len(Spaces())
	if got := len(SpacesLevel(-5)); got != 0 {
		t.Fatalf("negative level should give 0, got %d", got)
	}
	if got := len(SpacesLevel(1000)); got != full {
		t.Fatalf("overflow level should clamp to %d, got %d", full, got)
	}
	if got := len(SpacesLevel(2)); got != 2 {
		t.Fatalf("level 2 should give 2, got %d", got)
	}
}
