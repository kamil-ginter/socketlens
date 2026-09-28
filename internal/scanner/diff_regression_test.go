package scanner

import (
	"testing"

	"github.com/kamil-ginter/socketlens/internal/models"
)

func TestDiffComparableIgnoresPortsNotScannedTwice(t *testing.T) {
	previousOpen := []int{80, 8791}
	previousRequested := []int{80, 443, 8791}

	currentOpen := []models.PortResult{
		{Port: 443, Service: "HTTPS"},
		{Port: 8790, Service: "TCP/8790"},
	}
	currentRequested := []int{80, 443, 8790}

	newPorts, closedPorts, comparable := DiffComparable(
		previousOpen,
		previousRequested,
		currentOpen,
		currentRequested,
	)

	if comparable != 2 {
		t.Fatalf("expected 2 comparable ports, got %d", comparable)
	}

	if len(newPorts) != 1 || newPorts[0] != 443 {
		t.Fatalf("unexpected new ports: %v", newPorts)
	}

	if len(closedPorts) != 1 || closedPorts[0] != 80 {
		t.Fatalf("unexpected closed ports: %v", closedPorts)
	}
}

func TestDiffComparableWithDifferentPortSets(t *testing.T) {
	previousOpen := []int{8791}
	previousRequested := []int{8791}
	currentOpen := []models.PortResult{{Port: 8790, Service: "TCP/8790"}}
	currentRequested := []int{8790}

	newPorts, closedPorts, comparable := DiffComparable(
		previousOpen,
		previousRequested,
		currentOpen,
		currentRequested,
	)

	if comparable != 0 {
		t.Fatalf("expected 0 comparable ports, got %d", comparable)
	}
	if len(newPorts) != 0 || len(closedPorts) != 0 {
		t.Fatalf("expected no changes, got new=%v closed=%v", newPorts, closedPorts)
	}
}
