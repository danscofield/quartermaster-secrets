package auth_test

import (
	"testing"

	"github.com/qm-secrets/qm-secrets/internal/auth"
)

func TestIntersects(t *testing.T) {
	if !auth.Intersects([]string{"a", "b"}, []string{"b", "c"}) {
		t.Fatal("expected intersection")
	}
	if auth.Intersects([]string{"a"}, []string{"b"}) {
		t.Fatal("expected no intersection")
	}
}

func TestCanRead(t *testing.T) {
	owners := []string{"team-a"}
	retrievers := []string{"team-b"}

	if !auth.CanRead([]string{"team-a"}, owners, retrievers) {
		t.Fatal("owner should read")
	}
	if !auth.CanRead([]string{"team-b"}, owners, retrievers) {
		t.Fatal("retriever should read")
	}
	if auth.CanRead([]string{"team-c"}, owners, retrievers) {
		t.Fatal("unauthorized should not read")
	}
}

func TestCanUpdate(t *testing.T) {
	owners := []string{"team-a"}
	if !auth.CanUpdate([]string{"team-a"}, owners) {
		t.Fatal("owner should update")
	}
	if auth.CanUpdate([]string{"team-b"}, owners) {
		t.Fatal("non-owner should not update")
	}
}

func TestValidateOwnerIntersection(t *testing.T) {
	if !auth.ValidateOwnerIntersection([]string{"team-a"}, []string{"team-a", "team-b"}) {
		t.Fatal("expected valid owner intersection on create")
	}
	if auth.ValidateOwnerIntersection([]string{"team-c"}, []string{"team-a"}) {
		t.Fatal("expected no intersection")
	}
}
