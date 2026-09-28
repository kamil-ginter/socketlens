package scanner

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
)

func ResolvePrivateTarget(ctx context.Context, target string) (string, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", errors.New("target is required")
	}

	if strings.Contains(target, "://") || strings.ContainsAny(target, "/\\") {
		return "", "", errors.New("enter a hostname or IP address, not a URL or path")
	}

	normalized := strings.ToLower(target)

	if ip := net.ParseIP(target); ip != nil {
		if !allowedIP(ip) {
			return "", "", errors.New("SocketLens 0.1 only scans private or loopback addresses")
		}
		return normalized, ip.String(), nil
	}

	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, target)
	if err != nil {
		return "", "", fmt.Errorf("could not resolve target: %w", err)
	}

	for _, address := range addresses {
		if allowedIP(address.IP) {
			return normalized, address.IP.String(), nil
		}
	}

	return "", "", errors.New("target did not resolve to a private or loopback address")
}

func allowedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate()
}
