package store

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRecentPRFilterPersistence(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "heckl.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := Migrate(ctx, s.db); err != nil {
		t.Fatal(err)
	}

	value, err := s.GetRecentPRFilter(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if value != "" {
		t.Fatalf("initial filter = %q, want empty", value)
	}

	const saved = `{"organizations":["acme"],"repositories":["other/repo"]}`
	if err := s.SetRecentPRFilter(ctx, saved); err != nil {
		t.Fatal(err)
	}
	value, err = s.GetRecentPRFilter(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if value != saved {
		t.Fatalf("filter = %q, want %q", value, saved)
	}
}
