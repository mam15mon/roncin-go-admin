package syncrunner

import (
	"strings"
	"testing"
)

func TestOpenStoreRequiresDatabaseSource(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "")

	store, cleanup, err := OpenStore()
	if err == nil || !strings.Contains(err.Error(), "DATABASE_SOURCE") {
		t.Fatalf("OpenStore() error = %v, want DATABASE_SOURCE error", err)
	}
	if store != nil || cleanup != nil {
		t.Fatal("OpenStore() returned non-nil store or cleanup, want both nil")
	}
}
