package store_test

import (
	"aide/cli/internal/persistence/store"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestTeamRepo_Migration(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	if s.Team == nil {
		t.Fatal("store.Team is nil after Open")
	}

	members, err := s.Team.All()
	if err != nil {
		t.Fatalf("All() on empty table: %v", err)
	}
	if len(members) != 0 {
		t.Fatalf("expected 0 members, got %d", len(members))
	}
}

func TestTeamRepo_UpsertAndAll(t *testing.T) {
	s := openTestStore(t)

	members := []store.Member{
		{Name: "Alice", Email: "alice@example.com", Role: "Manager", Registration: "001", Source: "config"},
		{Name: "Bob", Email: "bob@example.com", Role: "Engineer", Registration: "002", Source: "config"},
	}

	if err := s.Team.Upsert(members); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	all, err := s.Team.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 members, got %d", len(all))
	}
}

func TestTeamRepo_Fingerprint(t *testing.T) {
	fp1 := store.MemberFingerprint("Alice", "001", "")
	fp2 := store.MemberFingerprint("Alice", "001", "other@example.com")
	fp3 := store.MemberFingerprint("Alice", "", "alice@example.com")
	fp4 := store.MemberFingerprint("Alice", "", "")

	if fp1 != fp2 {
		t.Error("registration takes priority over email: fp1 should equal fp2")
	}
	if fp1 == fp3 {
		t.Error("different registration and email should yield different fingerprints")
	}
	if fp3 == fp4 {
		t.Error("email vs no-email should yield different fingerprints")
	}
}

func TestTeamRepo_ManagerResolution(t *testing.T) {
	s := openTestStore(t)

	members := []store.Member{
		{Name: "Alice", Registration: "001", Source: "config"},
		{Name: "Bob", Registration: "002", Source: "config", ManagerRef: "Alice"},
	}

	if err := s.Team.Upsert(members); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	all, err := s.Team.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	var alice, bob *store.Member
	for i := range all {
		switch all[i].Name {
		case "Alice":
			alice = &all[i]
		case "Bob":
			bob = &all[i]
		}
	}

	if alice == nil || bob == nil {
		t.Fatal("expected both Alice and Bob in results")
	}
	if alice.ManagerID != nil {
		t.Errorf("Alice should have no manager, got %v", alice.ManagerID)
	}
	if bob.ManagerID == nil {
		t.Fatal("Bob should have a manager_id set")
	}
	if *bob.ManagerID != alice.ID {
		t.Errorf("Bob's manager_id should be Alice's id (%d), got %d", alice.ID, *bob.ManagerID)
	}
}

func TestTeamRepo_ManagerResolutionByRegistration(t *testing.T) {
	s := openTestStore(t)

	members := []store.Member{
		{Name: "Alice", Registration: "001", Source: "rh_management_portal"},
		{Name: "Bob", Registration: "002", Source: "rh_management_portal", ManagerRef: "001"},
	}

	if err := s.Team.Upsert(members); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	all, err := s.Team.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	var alice, bob *store.Member
	for i := range all {
		switch all[i].Name {
		case "Alice":
			alice = &all[i]
		case "Bob":
			bob = &all[i]
		}
	}

	if bob.ManagerID == nil {
		t.Fatal("Bob should have manager_id resolved via registration")
	}
	if *bob.ManagerID != alice.ID {
		t.Errorf("expected manager_id=%d, got %d", alice.ID, *bob.ManagerID)
	}
}

func TestTeamRepo_UpsertIdempotent(t *testing.T) {
	s := openTestStore(t)

	members := []store.Member{
		{Name: "Alice", Registration: "001", Role: "Engineer", Source: "config"},
	}

	if err := s.Team.Upsert(members); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	members[0].Role = "Senior Engineer"
	if err := s.Team.Upsert(members); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	all, err := s.Team.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 member after two upserts, got %d", len(all))
	}
	if all[0].Role != "Senior Engineer" {
		t.Errorf("expected updated role, got %q", all[0].Role)
	}
}

func TestTeamRepo_OrphanPass(t *testing.T) {
	s := openTestStore(t)

	first := []store.Member{
		{Name: "Alice", Registration: "001", Source: "rh_management_portal"},
		{Name: "Bob", Registration: "002", Source: "rh_management_portal", ManagerRef: "001"},
		{Name: "Carol", Registration: "003", Source: "rh_management_portal", ManagerRef: "001"},
	}
	if err := s.Team.Upsert(first); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	second := []store.Member{
		{Name: "Alice", Registration: "001", Source: "rh_management_portal"},
		{Name: "Bob", Registration: "002", Source: "rh_management_portal", ManagerRef: "001"},
	}
	if err := s.Team.Upsert(second); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	all, err := s.Team.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 members (Carol kept for history), got %d", len(all))
	}

	for _, m := range all {
		if m.Name == "Carol" && m.ManagerID != nil {
			t.Errorf("Carol should have manager_id = NULL after orphan pass, got %v", m.ManagerID)
		}
	}
}

func TestTeamRepo_ConfigSourceSkipsOrphan(t *testing.T) {
	s := openTestStore(t)

	first := []store.Member{
		{Name: "Alice", Registration: "001", Source: "config"},
		{Name: "Bob", Registration: "002", Source: "config", ManagerRef: "Alice"},
	}
	if err := s.Team.Upsert(first); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}

	second := []store.Member{
		{Name: "Alice", Registration: "001", Source: "config"},
	}
	if err := s.Team.Upsert(second); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}

	all, err := s.Team.All()
	if err != nil {
		t.Fatalf("All: %v", err)
	}

	var bob *store.Member
	for i := range all {
		if all[i].Name == "Bob" {
			bob = &all[i]
		}
	}
	if bob == nil {
		t.Fatal("Bob should still be in DB (config skips orphan pass)")
	}
	if bob.ManagerID == nil {
		t.Error("Bob's manager_id should still be set (config skips orphan pass)")
	}
}

func TestTeamRepo_Resolve(t *testing.T) {
	s := openTestStore(t)

	aliasesJSON, _ := json.Marshal([]string{"Alicia", "ali"})
	members := []store.Member{
		{Name: "Alice", Email: "alice@example.com", Registration: "001", Aliases: string(aliasesJSON), Source: "config"},
	}
	if err := s.Team.Upsert(members); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	tests := []struct{ input, want string }{
		{"Alice", "Alice"},
		{"alice@example.com", "Alice"},
		{"001", "Alice"},
		{"Alicia", "Alice"},
		{"ali", "Alice"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := s.Team.Resolve(tt.input)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMigration_TeamMembersTableExists(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "aide.db")

	s, err := store.Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.Close()

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("DB file not created: %v", err)
	}
}
