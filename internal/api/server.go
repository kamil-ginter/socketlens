package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kamil-ginter/socketlens/internal/models"
	"github.com/kamil-ginter/socketlens/internal/scanner"
	"github.com/kamil-ginter/socketlens/internal/store"
)

type Server struct {
	store     *store.Store
	uiHandler http.Handler
}

func New(db *store.Store, uiHandler http.Handler) *Server {
	return &Server{
		store:     db,
		uiHandler: uiHandler,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("POST /api/scan", s.scan)
	mux.HandleFunc("GET /api/history", s.history)
	mux.Handle("/", s.uiHandler)

	return securityHeaders(mux)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"service": "socketlens",
		"status":  "healthy",
		"version": "0.1.0",
		"time":    time.Now().UTC(),
	})
}

func (s *Server) scan(w http.ResponseWriter, r *http.Request) {
	var request models.ScanRequest

	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024*1024))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	timeoutMs := request.TimeoutMs
	if timeoutMs == 0 {
		timeoutMs = 400
	}
	if timeoutMs < 100 || timeoutMs > 5000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "timeoutMs must be between 100 and 5000"})
		return
	}

	ports, err := scanner.ParsePorts(request.Ports)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	resolveContext, cancelResolve := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancelResolve()

	normalizedTarget, resolvedIP, err := scanner.ResolvePrivateTarget(resolveContext, request.Target)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	previous, hasPrevious, err := s.store.LatestSnapshot(normalizedTarget)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not read previous scan"})
		return
	}

	scanContext, cancelScan := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancelScan()

	started := time.Now().UTC()
	openPorts, durationMs := scanner.Scan(
		scanContext,
		resolvedIP,
		ports,
		time.Duration(timeoutMs)*time.Millisecond,
	)

	newPorts := []int{}
	closedPorts := []int{}
	hasComparableBaseline := false

	if hasPrevious {
		var comparable int
		newPorts, closedPorts, comparable = scanner.DiffComparable(
			previous.OpenPorts,
			previous.RequestedPorts,
			openPorts,
			ports,
		)
		hasComparableBaseline = comparable > 0
	}

	result := models.ScanResult{
		Target:         normalizedTarget,
		ResolvedIP:     resolvedIP,
		StartedAt:      started,
		DurationMs:     durationMs,
		RequestedPorts: len(ports),
		OpenPorts:      openPorts,
		NewPorts:       newPorts,
		ClosedPorts:    closedPorts,
		HasPrevious:    hasComparableBaseline,
	}

	if err := s.store.SaveScan(&result); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save scan"})
		return
	}

	if err := s.store.SaveRequestedPorts(result.ID, ports); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not save requested ports"})
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	limit := 20

	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a number"})
			return
		}
		limit = parsed
	}

	items, err := s.store.History(limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "could not load history"})
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; script-src 'self'; style-src 'self'; connect-src 'self'; img-src 'self' data:;",
		)

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
