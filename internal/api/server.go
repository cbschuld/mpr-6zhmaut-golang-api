package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/amp"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/config"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/model"
	"github.com/cbschuld/mpr-6zhmaut-golang-api/internal/serial"
)

// Server is the HTTP API server.
type Server struct {
	cfg        *config.Config
	controller *amp.Controller
	cache      *amp.ZoneCache
	state      *amp.StateMachine
	queue      *serial.Queue
	events     *amp.EventLog
	health     *amp.HealthMonitor
	port       *serial.Port
	logger     *slog.Logger
	startTime  time.Time
	mux        *http.ServeMux
	webFS      fs.FS // embedded web UI assets
}

// NewServer creates a new API server.
func NewServer(
	cfg *config.Config,
	controller *amp.Controller,
	cache *amp.ZoneCache,
	state *amp.StateMachine,
	queue *serial.Queue,
	events *amp.EventLog,
	health *amp.HealthMonitor,
	port *serial.Port,
	logger *slog.Logger,
	webFS fs.FS,
) *Server {
	s := &Server{
		cfg:        cfg,
		controller: controller,
		cache:      cache,
		state:      state,
		queue:      queue,
		events:     events,
		health:     health,
		port:       port,
		logger:     logger,
		startTime:  time.Now(),
		mux:        http.NewServeMux(),
		webFS:      webFS,
	}
	s.routes()
	return s
}

// Handler returns the HTTP handler.
func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	if s.cfg.CORS {
		handler = corsMiddleware(handler)
	}
	handler = s.loggingMiddleware(handler)
	return handler
}

func (s *Server) routes() {
	// API routes (matched first due to specificity)
	s.mux.HandleFunc("GET /api/zones", s.handleGetZones)
	s.mux.HandleFunc("GET /api/zones/{zone}", s.handleGetZone)
	s.mux.HandleFunc("GET /api/zones/{zone}/{attribute}", s.handleGetZoneAttribute)
	s.mux.HandleFunc("POST /api/zones/{zone}/{attribute}", s.handleSetZoneAttribute)
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/health/events", s.handleHealthEvents)

	// Legacy routes (without /api prefix) for backward compatibility
	s.mux.HandleFunc("GET /zones", s.handleGetZones)
	s.mux.HandleFunc("GET /zones/{zone}", s.handleGetZone)
	s.mux.HandleFunc("GET /zones/{zone}/{attribute}", s.handleGetZoneAttribute)
	s.mux.HandleFunc("POST /zones/{zone}/{attribute}", s.handleSetZoneAttribute)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /health/events", s.handleHealthEvents)

	// Web UI: serve embedded static files, fallback to index.html for SPA routing
	if s.webFS != nil {
		s.mux.Handle("GET /", s.spaHandler())
	}
}

// spaHandler serves static files from the embedded FS, falling back to
// index.html for client-side routing (React Router, etc.).
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.webFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else {
			path = strings.TrimPrefix(path, "/")
		}

		// Check if file exists in the embedded FS
		if _, err := fs.Stat(s.webFS, path); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found -- serve index.html for SPA client-side routing
		indexFile, err := fs.ReadFile(s.webFS, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexFile)
	})
}

func (s *Server) handleGetZones(w http.ResponseWriter, r *http.Request) {
	live := r.URL.Query().Get("live") == "true"

	if live {
		if !s.state.IsReady() {
			writeError(w, http.StatusServiceUnavailable, "amp connection recovering")
			return
		}
		zones, err := s.controller.QueryAllZones(r.Context())
		if err != nil {
			s.handleAmpError(w, err)
			return
		}
		sortZones(zones)
		writeJSON(w, http.StatusOK, zones)
		return
	}

	zones := s.cache.GetAll()
	sortZones(zones)
	writeJSON(w, http.StatusOK, zones)
}

func (s *Server) handleGetZone(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zone")
	if !serial.ValidZoneID(zoneID, s.cfg.AmpCount) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s is not a valid zone", zoneID))
		return
	}

	live := r.URL.Query().Get("live") == "true"

	if live {
		if !s.state.IsReady() {
			writeError(w, http.StatusServiceUnavailable, "amp connection recovering")
			return
		}
		zone, err := s.controller.QueryZone(r.Context(), zoneID)
		if err != nil {
			s.handleAmpError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, zone)
		return
	}

	zone, ok := s.cache.Get(zoneID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("zone %s not in cache", zoneID))
		return
	}
	writeJSON(w, http.StatusOK, zone)
}

func (s *Server) handleGetZoneAttribute(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zone")
	attrName := r.PathValue("attribute")

	if !serial.ValidZoneID(zoneID, s.cfg.AmpCount) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s is not a valid zone", zoneID))
		return
	}

	attr, ok := serial.ResolveAttribute(attrName)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s is not a valid attribute", attrName))
		return
	}

	live := r.URL.Query().Get("live") == "true"

	if live {
		if !s.state.IsReady() {
			writeError(w, http.StatusServiceUnavailable, "amp connection recovering")
			return
		}
		zone, err := s.controller.QueryZone(r.Context(), zoneID)
		if err != nil {
			s.handleAmpError(w, err)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(zone.GetAttribute(attr)))
		return
	}

	zone, ok := s.cache.Get(zoneID)
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("zone %s not in cache", zoneID))
		return
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(zone.GetAttribute(attr)))
}

func (s *Server) handleSetZoneAttribute(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("zone")
	attrName := r.PathValue("attribute")

	if !serial.ValidZoneID(zoneID, s.cfg.AmpCount) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s is not a valid zone", zoneID))
		return
	}

	attr, ok := serial.ResolveAttribute(attrName)
	if !ok {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s is not a valid attribute", attrName))
		return
	}

	if !s.state.IsReady() {
		writeError(w, http.StatusServiceUnavailable, "amp connection recovering")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	value := strings.TrimSpace(string(body))

	zone, err := s.controller.SetAttribute(r.Context(), zoneID, attr, value)
	if err != nil {
		s.handleAmpError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, zone)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	lastCheck, lastCheckOk := s.health.LastCheck()
	qStats := s.queue.GetStats()
	rStats := s.controller.GetRecoveryStats()

	status := "ready"
	if !s.state.IsReady() {
		status = strings.ToLower(s.state.Current().String())
	}

	resp := map[string]any{
		"status":         status,
		"uptime_seconds": int(time.Since(s.startTime).Seconds()),
		"serial": map[string]any{
			"device":            s.cfg.Device,
			"current_baud_rate": s.port.BaudRate(),
			"target_baud_rate":  s.cfg.TargetBaudRate,
		},
		"state_machine": map[string]any{
			"state":                 s.state.Current().String(),
			"last_transition":       s.state.LastTransition(),
			"time_in_state_seconds": int(s.state.TimeInState().Seconds()),
		},
		"cache": map[string]any{
			"zone_count":    s.cache.Count(),
			"last_poll":     s.cache.LastUpdate(),
			"cache_age_ms":  s.cache.Age().Milliseconds(),
			"poll_interval": s.cfg.PollInterval.String(),
		},
		"queue": map[string]any{
			"pending_commands":    qStats.Pending,
			"total_commands_sent": qStats.TotalCommands,
			"total_timeouts":      qStats.TotalTimeouts,
			"total_errors":        qStats.TotalErrors,
		},
		"recovery": rStats,
		"health_check": map[string]any{
			"last_check":    lastCheck,
			"last_check_ok": lastCheckOk,
			"interval":      s.health.Interval().String(),
		},
		"amps": map[string]any{
			"count": s.cfg.AmpCount,
		},
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHealthEvents(w http.ResponseWriter, r *http.Request) {
	events := s.events.All()
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) handleAmpError(w http.ResponseWriter, err error) {
	switch {
	case err == serial.ErrRecovering:
		writeError(w, http.StatusServiceUnavailable, "amp connection recovering")
	case err == serial.ErrTimeout:
		writeError(w, http.StatusGatewayTimeout, "amp response timeout")
	default:
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		s.logger.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"live", r.URL.Query().Get("live") == "true",
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func sortZones(zones []model.Zone) {
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].Zone < zones[j].Zone
	})
}
