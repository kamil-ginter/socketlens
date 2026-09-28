package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/kamil-ginter/socketlens/internal/models"
)

func TestSaveScanHistoryAndPreviousPorts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "socketlens.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	previous, found, err := store.LatestOpenPorts("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if found || len(previous) != 0 {
		t.Fatalf("expected no previous scan, got found=%v ports=%v", found, previous)
	}

	result := models.ScanResult{
		Target:         "127.0.0.1",
		ResolvedIP:     "127.0.0.1",
		StartedAt:      time.Now().UTC(),
		DurationMs:     10,
		RequestedPorts: 2,
		OpenPorts: []models.PortResult{
			{Port: 80, Service: "HTTP"},
			{Port: 443, Service: "HTTPS"},
		},
		NewPorts:    []int{},
		ClosedPorts: []int{},
		HasPrevious: false,
	}

	if err := store.SaveScan(&result); err != nil {
		t.Fatal(err)
	}
	if result.ID == 0 {
		t.Fatal("expected scan ID")
	}

	previous, found, err = store.LatestOpenPorts("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !found || len(previous) != 2 {
		t.Fatalf("expected 2 previous ports, got found=%v ports=%v", found, previous)
	}

	history, err := store.History(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].OpenCount != 2 {
		t.Fatalf("unexpected history: %#v", history)
	}
}
