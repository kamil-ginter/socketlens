package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kamil-ginter/socketlens/internal/models"
)

func TestSnapshotStoresRequestedPortSet(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "snapshot.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	result := models.ScanResult{
		Target:         "127.0.0.1",
		ResolvedIP:     "127.0.0.1",
		StartedAt:      time.Now().UTC(),
		DurationMs:     1,
		RequestedPorts: 3,
		OpenPorts: []models.PortResult{
			{Port: 80, Service: "HTTP"},
		},
	}

	if err := store.SaveScan(&result); err != nil {
		t.Fatal(err)
	}

	if err := store.SaveRequestedPorts(result.ID, []int{80, 443, 8790}); err != nil {
		t.Fatal(err)
	}

	snapshot, found, err := store.LatestSnapshot("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected snapshot")
	}

	if len(snapshot.OpenPorts) != 1 || snapshot.OpenPorts[0] != 80 {
		t.Fatalf("unexpected open ports: %v", snapshot.OpenPorts)
	}

	expected := []int{80, 443, 8790}
	if len(snapshot.RequestedPorts) != len(expected) {
		t.Fatalf("unexpected requested ports: %v", snapshot.RequestedPorts)
	}
	for index := range expected {
		if snapshot.RequestedPorts[index] != expected[index] {
			t.Fatalf("unexpected requested ports: %v", snapshot.RequestedPorts)
		}
	}
}
