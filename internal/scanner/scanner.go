package scanner

import (
	"context"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/kamil-ginter/socketlens/internal/models"
)

const maxWorkers = 64

func Scan(ctx context.Context, ip string, ports []int, timeout time.Duration) ([]models.PortResult, int64) {
	started := time.Now()

	workers := maxWorkers
	if len(ports) < workers {
		workers = len(ports)
	}
	if workers < 1 {
		return []models.PortResult{}, 0
	}

	jobs := make(chan int)
	results := make(chan models.PortResult, len(ports))
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()

		for port := range jobs {
			select {
			case <-ctx.Done():
				return
			default:
			}

			address := net.JoinHostPort(ip, fmt.Sprintf("%d", port))
			connection, err := net.DialTimeout("tcp", address, timeout)
			if err != nil {
				continue
			}

			_ = connection.Close()

			results <- models.PortResult{
				Port:    port,
				Service: ServiceName(port),
			}
		}
	}

	wg.Add(workers)
	for index := 0; index < workers; index++ {
		go worker()
	}

	go func() {
		defer close(jobs)
		for _, port := range ports {
			select {
			case <-ctx.Done():
				return
			case jobs <- port:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	open := make([]models.PortResult, 0)
	for result := range results {
		open = append(open, result)
	}

	sort.Slice(open, func(i, j int) bool {
		return open[i].Port < open[j].Port
	})

	return open, time.Since(started).Milliseconds()
}

func Diff(previous []int, current []models.PortResult) ([]int, []int) {
	previousSet := map[int]bool{}
	currentSet := map[int]bool{}

	for _, port := range previous {
		previousSet[port] = true
	}

	for _, item := range current {
		currentSet[item.Port] = true
	}

	newPorts := make([]int, 0)
	closedPorts := make([]int, 0)

	for port := range currentSet {
		if !previousSet[port] {
			newPorts = append(newPorts, port)
		}
	}

	for port := range previousSet {
		if !currentSet[port] {
			closedPorts = append(closedPorts, port)
		}
	}

	sort.Ints(newPorts)
	sort.Ints(closedPorts)

	return newPorts, closedPorts
}

func DiffComparable(
	previousOpen []int,
	previousRequested []int,
	currentOpen []models.PortResult,
	currentRequested []int,
) ([]int, []int, int) {
	previousOpenSet := intSet(previousOpen)
	previousRequestedSet := intSet(previousRequested)
	currentRequestedSet := intSet(currentRequested)

	currentOpenSet := map[int]bool{}
	for _, item := range currentOpen {
		currentOpenSet[item.Port] = true
	}

	comparable := 0
	for port := range currentRequestedSet {
		if previousRequestedSet[port] {
			comparable++
		}
	}

	newPorts := make([]int, 0)
	closedPorts := make([]int, 0)

	for port := range currentOpenSet {
		if previousRequestedSet[port] && !previousOpenSet[port] {
			newPorts = append(newPorts, port)
		}
	}

	for port := range previousOpenSet {
		if currentRequestedSet[port] && !currentOpenSet[port] {
			closedPorts = append(closedPorts, port)
		}
	}

	sort.Ints(newPorts)
	sort.Ints(closedPorts)

	return newPorts, closedPorts, comparable
}

func intSet(values []int) map[int]bool {
	result := make(map[int]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func ServiceName(port int) string {
	services := map[int]string{
		20:    "FTP data",
		21:    "FTP",
		22:    "SSH",
		23:    "Telnet",
		25:    "SMTP",
		53:    "DNS",
		80:    "HTTP",
		110:   "POP3",
		135:   "MS RPC",
		139:   "NetBIOS",
		143:   "IMAP",
		443:   "HTTPS",
		445:   "SMB",
		993:   "IMAPS",
		995:   "POP3S",
		1433:  "SQL Server",
		3000:  "Dev HTTP",
		3306:  "MySQL",
		3389:  "RDP",
		5432:  "PostgreSQL",
		6379:  "Redis",
		8080:  "HTTP alt",
		8443:  "HTTPS alt",
		27017: "MongoDB",
	}

	if name, ok := services[port]; ok {
		return name
	}

	return fmt.Sprintf("TCP/%d", port)
}
