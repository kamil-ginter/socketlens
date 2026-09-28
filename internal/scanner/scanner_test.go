package scanner

import (
	"net"
	"testing"

	"github.com/kamil-ginter/socketlens/internal/models"
)

func TestParsePorts(t *testing.T) {
	ports, err := ParsePorts("22,80,443,8000-8002,80")
	if err != nil {
		t.Fatal(err)
	}

	expected := []int{22, 80, 443, 8000, 8001, 8002}
	if len(ports) != len(expected) {
		t.Fatalf("expected %d ports, got %d", len(expected), len(ports))
	}

	for index := range expected {
		if ports[index] != expected[index] {
			t.Fatalf("expected port %d at index %d, got %d", expected[index], index, ports[index])
		}
	}
}

func TestParsePortsRejectsInvalidRanges(t *testing.T) {
	if _, err := ParsePorts("100-10"); err == nil {
		t.Fatal("expected descending range to fail")
	}

	if _, err := ParsePorts("0"); err == nil {
		t.Fatal("expected port 0 to fail")
	}
}

func TestAllowedIP(t *testing.T) {
	if !allowedIP(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback should be allowed")
	}
	if !allowedIP(net.ParseIP("192.168.1.10")) {
		t.Fatal("private IPv4 should be allowed")
	}
	if allowedIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("public IPv4 should not be allowed")
	}
}

func TestDiff(t *testing.T) {
	previous := []int{22, 80, 443}
	current := []models.PortResult{
		{Port: 22, Service: "SSH"},
		{Port: 443, Service: "HTTPS"},
		{Port: 8080, Service: "HTTP alt"},
	}

	newPorts, closedPorts := Diff(previous, current)

	if len(newPorts) != 1 || newPorts[0] != 8080 {
		t.Fatalf("unexpected new ports: %v", newPorts)
	}
	if len(closedPorts) != 1 || closedPorts[0] != 80 {
		t.Fatalf("unexpected closed ports: %v", closedPorts)
	}
}
