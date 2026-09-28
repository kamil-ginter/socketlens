package models

import "time"

type PortResult struct {
	Port    int    `json:"port"`
	Service string `json:"service"`
}

type ScanRequest struct {
	Target    string `json:"target"`
	Ports     string `json:"ports"`
	TimeoutMs int    `json:"timeoutMs"`
}

type ScanResult struct {
	ID             int64        `json:"id"`
	Target         string       `json:"target"`
	ResolvedIP     string       `json:"resolvedIp"`
	StartedAt      time.Time    `json:"startedAt"`
	DurationMs     int64        `json:"durationMs"`
	RequestedPorts int          `json:"requestedPorts"`
	OpenPorts      []PortResult `json:"openPorts"`
	NewPorts       []int        `json:"newPorts"`
	ClosedPorts    []int        `json:"closedPorts"`
	HasPrevious    bool         `json:"hasPrevious"`
}

type HistoryItem struct {
	ID         int64     `json:"id"`
	Target     string    `json:"target"`
	ResolvedIP string    `json:"resolvedIp"`
	StartedAt  time.Time `json:"startedAt"`
	DurationMs int64     `json:"durationMs"`
	OpenCount  int       `json:"openCount"`
}
