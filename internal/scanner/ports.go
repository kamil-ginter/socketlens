package scanner

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const maxPorts = 1024

func ParsePorts(value string) ([]int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("ports are required")
	}

	seen := map[int]bool{}
	ports := make([]int, 0)

	add := func(port int) error {
		if port < 1 || port > 65535 {
			return fmt.Errorf("port %d is outside 1-65535", port)
		}
		if seen[port] {
			return nil
		}
		if len(ports) >= maxPorts {
			return fmt.Errorf("a scan can contain at most %d unique ports", maxPorts)
		}
		seen[port] = true
		ports = append(ports, port)
		return nil
	}

	for _, token := range strings.Split(value, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		if strings.Contains(token, "-") {
			parts := strings.Split(token, "-")
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid port range %q", token)
			}

			start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid port range %q", token)
			}
			end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid port range %q", token)
			}
			if start > end {
				return nil, fmt.Errorf("invalid descending port range %q", token)
			}

			for port := start; port <= end; port++ {
				if err := add(port); err != nil {
					return nil, err
				}
			}
			continue
		}

		port, err := strconv.Atoi(token)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q", token)
		}
		if err := add(port); err != nil {
			return nil, err
		}
	}

	if len(ports) == 0 {
		return nil, errors.New("no ports were provided")
	}

	sort.Ints(ports)
	return ports, nil
}
