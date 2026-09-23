package infrastructure_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"traineebox/internal/tickets/infrastructure"
)

func catalogDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// .../traineebox/internal/tickets/infrastructure -> .../emergency/artifacts/etc/traineebox/catalog
	dir := filepath.Clean(filepath.Join(
		filepath.Dir(file), "..", "..", "..", "..",
		"artifacts", "etc", "traineebox", "catalog",
	))
	return dir
}

func TestLoadCatalog(t *testing.T) {
	cat, err := infrastructure.LoadCatalog(catalogDir(t))
	if err != nil {
		t.Fatal(err)
	}
	repo := infrastructure.NewCatalogRepository(cat)
	types, err := repo.ListIncidentTypes(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(types) < 40 {
		t.Fatalf("types = %d", len(types))
	}
	fire, err := repo.FindIncidentTypeByCode(t.Context(), "101")
	if err != nil {
		t.Fatal(err)
	}
	if fire.Code != "101" {
		t.Fatalf("code = %s", fire.Code)
	}
	groups, err := repo.ListTagGroupsByType(t.Context(), "101")
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) == 0 {
		t.Fatal("expected fire groups")
	}
	svcs, err := repo.ListServices(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs) < 100 {
		t.Fatalf("services = %d", len(svcs))
	}
	ok, err := repo.ServiceExists(t.Context(), []string{"sluzhba_101"})
	if err != nil || !ok {
		t.Fatal("sluzhba_101 missing")
	}
}

func TestRecommendServices(t *testing.T) {
	cat, err := infrastructure.LoadCatalog(catalogDir(t))
	if err != nil {
		t.Fatal(err)
	}
	repo := infrastructure.NewCatalogRepository(cat)
	got, err := repo.RecommendServices(t.Context(), "101", []string{"threat_people_yes", "medical_help_yes"})
	if err != nil {
		t.Fatal(err)
	}
	need := []string{"sluzhba_101", "tsemp", "sluzhba_103"}
	set := make(map[string]bool, len(got))
	for _, code := range got {
		set[code] = true
	}
	for _, code := range need {
		if !set[code] {
			t.Fatalf("missing %s in %v", code, got)
		}
	}
}
