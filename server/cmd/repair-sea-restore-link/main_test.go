package main

import (
	"testing"

	"github.com/google/uuid"
)

func TestRepairInputRequiresExplicitTarget(t *testing.T) {
	id := uuid.NewString()
	if _, err := parseInput(id, id, id, id, 7, "显式修复"); err != nil {
		t.Fatal(err)
	}
	for _, in := range []struct {
		org     string
		version uint64
		reason  string
	}{{"", 7, "原因"}, {id, 0, "原因"}, {id, 7, ""}} {
		if _, err := parseInput(in.org, id, id, id, in.version, in.reason); err == nil {
			t.Fatal("缺目标、版本或原因应拒绝")
		}
	}
}
