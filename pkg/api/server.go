package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/time/rate"
)

// DatabasePinger is an interface for checking database health
type DatabasePinger interface {
	Ping() error
}

// Server wraps the HTTP API with all service interfaces
type Server struct {
	router         *chi.Mux
	uiAlchemy      UIAlchemyService
	gpu            GPUService
	vqc            VQCService
	survivalGarden SurvivalGardenService
	db             DatabasePinger // Database for health checks
	port           string
	server         *http.Server
	startTime      time.Time // Server start time for uptime tracking

	// Security settings
	allowedOrigins  []string
	rateLimitPerMin int

	// Rate limiting state (per IP)
	rateLimiters sync.Map // map[string]*rate.Limiter

	// Metrics counters (thread-safe with atomic operations)
	counters struct {
		requestsTotal    int64 // Counter for total HTTP requests
		gpuOperations    int64 // Counter for GPU acceleration calls
		vqcOptimizations int64 // Counter for VQC optimization calls
		rateLimitedTotal int64 // Counter for rate-limited requests
		healthChecks     int64 // Counter for health check requests
		readyChecks      int64 // Counter for readiness check requests
		liveChecks       int64 // Counter for liveness check requests
	}
}

// NewServer creates a new API server instance
func NewServer(port string, allowedOrigins string, rateLimitPerMin int) *Server {
	// Parse allowed origins
	origins := []string{}
	if allowedOrigins != "" {
		for _, origin := range strings.Split(allowedOrigins, ",") {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				origins = append(origins, trimmed)
			}
		}
	}

	// Default to localhost only if no origins specified
	if len(origins) == 0 {
		origins = []string{
			"http://localhost:5173",
			"http://localhost:34115",
			"http://localhost:8080",
			"http://127.0.0.1:5173",
			"http://127.0.0.1:34115",
			"http://127.0.0.1:8080",
		}
	}

	s := &Server{
		router:          chi.NewRouter(),
		port:            port,
		allowedOrigins:  origins,
		rateLimitPerMin: rateLimitPerMin,
		startTime:       time.Now(), // Track server start time
	}

	// Middleware
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Timeout(60 * time.Second))

	// Rate limiting middleware (applied first to prevent abuse)
	s.router.Use(s.rateLimitMiddleware)

	// Metrics middleware - count all requests
	s.router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt64(&s.counters.requestsTotal, 1)
			next.ServeHTTP(w, r)
		})
	})

	// Secure CORS middleware (configurable origins, no wildcards!)
	s.router.Use(s.corsMiddleware)

	// Routes
	s.setupRoutes()

	return s
}

// SetUIAlchemyService injects the UI Alchemy implementation
func (s *Server) SetUIAlchemyService(svc UIAlchemyService) {
	s.uiAlchemy = svc
}

// SetGPUService injects the GPU implementation
func (s *Server) SetGPUService(svc GPUService) {
	s.gpu = svc
}

// SetVQCService injects the VQC implementation
func (s *Server) SetVQCService(svc VQCService) {
	s.vqc = svc
}

// SetSurvivalGardenService injects the survival garden implementation
func (s *Server) SetSurvivalGardenService(svc SurvivalGardenService) {
	s.survivalGarden = svc
}

// SetDatabase injects the database for health checks
func (s *Server) SetDatabase(db DatabasePinger) {
	s.db = db
}

// setupRoutes defines all API endpoints
func (s *Server) setupRoutes() {
	// Health check endpoints (Kubernetes/Docker ready!)
	s.router.Get("/health", s.healthCheck)   // Basic health check
	s.router.Get("/ready", s.readinessCheck) // Readiness probe (DB + services)
	s.router.Get("/live", s.livenessCheck)   // Liveness probe (no heavy ops)
	s.router.Get("/metrics", s.metrics)

	// API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		// UI Alchemy endpoints
		r.Route("/screens", func(r chi.Router) {
			r.Get("/{screenID}", s.getScreen)
			r.Post("/generate", s.generateScreen)
		})

		// GPU endpoints
		r.Route("/gpu", func(r chi.Router) {
			r.Post("/slerp", s.computeSLERP)
			r.Get("/status", s.gpuStatus)
			r.Post("/quaternion/multiply", s.quaternionMultiply)
			r.Post("/quaternion/normalize", s.quaternionNormalize)
		})

		// VQC endpoints
		r.Route("/vqc", func(r chi.Router) {
			r.Post("/optimize", s.vqcOptimize)
			r.Post("/classify", s.vqcClassify)
			r.Post("/climate", s.vqcClimate)
		})

		// Survival Garden simulation endpoints
		r.Route("/simulation", func(r chi.Router) {
			r.Post("/survival-garden", s.simulateSurvivalGarden)
			r.Get("/equilibrium", s.getEquilibrium)
		})

		// Visual Regime endpoints (fix global state!)
		r.Route("/regimes", func(r chi.Router) {
			r.Get("/", s.listRegimes)
			r.Get("/{regimeName}", s.getRegime)
			r.Post("/compute", s.computeRegime)
		})
	})
}

// ============================================================================
// SECURITY MIDDLEWARE
// ============================================================================

// rateLimitMiddleware enforces per-IP rate limiting
func (s *Server) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get client IP (use X-Forwarded-For if behind proxy, otherwise RemoteAddr)
		ip := r.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = r.Header.Get("X-Real-IP")
		}
		if ip == "" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			} else {
				ip = host
			}
		}

		// Get or create rate limiter for this IP
		limiterInterface, _ := s.rateLimiters.LoadOrStore(ip, rate.NewLimiter(
			rate.Limit(float64(s.rateLimitPerMin)/60.0), // Requests per second
			s.rateLimitPerMin, // Burst capacity
		))
		limiter := limiterInterface.(*rate.Limiter)

		// Check rate limit
		if !limiter.Allow() {
			atomic.AddInt64(&s.counters.rateLimitedTotal, 1)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware implements secure CORS with configurable allowed origins
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Check if origin is in allowed list
		allowed := false
		for _, allowedOrigin := range s.allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		// Set CORS headers only if origin is allowed
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")
		}

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			if allowed {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("🚀 API Server starting on port %s", s.port)
	log.Printf("📦 Bezos Mandate ENFORCED: All services externalized!")
	log.Printf("🔒 Security: CORS restricted to %d allowed origins", len(s.allowedOrigins))
	log.Printf("🛡️  Security: Rate limiting enabled (%d req/min per IP)", s.rateLimitPerMin)
	log.Printf("🔗 Try: curl http://localhost:%s/health", s.port)

	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// ============================================================================
// HEALTH CHECK ENDPOINTS (Kubernetes/Docker Ready!)
// ============================================================================

// healthCheck endpoint - Basic health check (always returns 200 if server running)
// Use: Kubernetes startupProbe, Docker HEALTHCHECK
func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&s.counters.healthChecks, 1)

	uptime := time.Since(s.startTime)

	resp := APIResponse{
		Version: "v1",
		Data: map[string]any{
			"status":         "ok",
			"timestamp":      time.Now().Format(time.RFC3339),
			"uptime_seconds": uptime.Seconds(),
			"uptime_human":   uptime.String(),
			"checks": map[string]string{
				"server": "running",
			},
		},
	}

	log.Printf("✓ Health check passed (uptime: %s)", uptime.String())
	writeJSON(w, http.StatusOK, resp)
}

// readinessCheck endpoint - Readiness probe (database connected, services initialized)
// Use: Kubernetes readinessProbe (determines if pod can receive traffic)
// Returns 503 if not ready, 200 if ready
func (s *Server) readinessCheck(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&s.counters.readyChecks, 1)

	checks := make(map[string]any)
	ready := true

	// Check 1: Database connectivity (CRITICAL)
	if s.db != nil {
		if err := s.db.Ping(); err != nil {
			checks["database"] = map[string]any{
				"status": "fail",
				"error":  err.Error(),
			}
			ready = false
			log.Printf("✗ Readiness check FAILED: database ping error: %v", err)
		} else {
			checks["database"] = map[string]any{
				"status": "ok",
			}
		}
	} else {
		checks["database"] = map[string]any{
			"status": "not_configured",
		}
		ready = false
		log.Printf("✗ Readiness check FAILED: database not configured")
	}

	// Check 2: Service initialization (verify injected services)
	servicesReady := true
	serviceStatus := make(map[string]string)

	if s.uiAlchemy == nil {
		serviceStatus["ui_alchemy"] = "not_initialized"
		servicesReady = false
	} else {
		serviceStatus["ui_alchemy"] = "ok"
	}

	if s.gpu == nil {
		serviceStatus["gpu"] = "not_initialized"
		servicesReady = false
	} else {
		serviceStatus["gpu"] = "ok"
	}

	if s.vqc == nil {
		serviceStatus["vqc"] = "not_initialized"
		servicesReady = false
	} else {
		serviceStatus["vqc"] = "ok"
	}

	if s.survivalGarden == nil {
		serviceStatus["survival_garden"] = "not_initialized"
		servicesReady = false
	} else {
		serviceStatus["survival_garden"] = "ok"
	}

	checks["services"] = serviceStatus

	if !servicesReady {
		ready = false
		log.Printf("✗ Readiness check FAILED: some services not initialized")
	}

	// Determine HTTP status code
	status := http.StatusOK
	overallStatus := "ready"
	if !ready {
		status = http.StatusServiceUnavailable
		overallStatus = "not_ready"
	}

	resp := APIResponse{
		Version: "v1",
		Data: map[string]any{
			"status":    overallStatus,
			"timestamp": time.Now().Format(time.RFC3339),
			"checks":    checks,
		},
	}

	if ready {
		log.Printf("✓ Readiness check passed (database + services OK)")
	}

	writeJSON(w, status, resp)
}

// livenessCheck endpoint - Liveness probe (app responsive, not deadlocked)
// Use: Kubernetes livenessProbe (determines if pod should be restarted)
// This should be FAST (no database calls, no heavy operations)
func (s *Server) livenessCheck(w http.ResponseWriter, r *http.Request) {
	atomic.AddInt64(&s.counters.liveChecks, 1)

	// Simple goroutine responsiveness check
	// If we can respond, we're alive (no deadlock)
	numGoroutines := goruntime.NumGoroutine()

	resp := APIResponse{
		Version: "v1",
		Data: map[string]any{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
			"checks": map[string]any{
				"goroutines": numGoroutines,
				"responsive": true,
			},
		},
	}

	// Log only if goroutine count is suspiciously high (potential leak)
	if numGoroutines > 1000 {
		log.Printf("⚠ Liveness check: high goroutine count (%d)", numGoroutines)
	}

	writeJSON(w, http.StatusOK, resp)
}

// metrics endpoint (Prometheus-compatible)
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	// Return Prometheus text format for scraping
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Write metrics in Prometheus exposition format
	fmt.Fprintf(w, "# HELP asymmetrica_requests_total Total HTTP requests processed\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_requests_total counter\n")
	fmt.Fprintf(w, "asymmetrica_requests_total %d\n\n", atomic.LoadInt64(&s.counters.requestsTotal))

	fmt.Fprintf(w, "# HELP asymmetrica_gpu_operations_total GPU acceleration operations\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_gpu_operations_total counter\n")
	fmt.Fprintf(w, "asymmetrica_gpu_operations_total %d\n\n", atomic.LoadInt64(&s.counters.gpuOperations))

	fmt.Fprintf(w, "# HELP asymmetrica_vqc_optimizations_total VQC optimizations run\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_vqc_optimizations_total counter\n")
	fmt.Fprintf(w, "asymmetrica_vqc_optimizations_total %d\n\n", atomic.LoadInt64(&s.counters.vqcOptimizations))

	fmt.Fprintf(w, "# HELP asymmetrica_rate_limited_total Requests blocked by rate limiting\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_rate_limited_total counter\n")
	fmt.Fprintf(w, "asymmetrica_rate_limited_total %d\n\n", atomic.LoadInt64(&s.counters.rateLimitedTotal))

	fmt.Fprintf(w, "# HELP asymmetrica_health_checks_total Basic health check requests\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_health_checks_total counter\n")
	fmt.Fprintf(w, "asymmetrica_health_checks_total %d\n\n", atomic.LoadInt64(&s.counters.healthChecks))

	fmt.Fprintf(w, "# HELP asymmetrica_ready_checks_total Readiness check requests\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_ready_checks_total counter\n")
	fmt.Fprintf(w, "asymmetrica_ready_checks_total %d\n\n", atomic.LoadInt64(&s.counters.readyChecks))

	fmt.Fprintf(w, "# HELP asymmetrica_live_checks_total Liveness check requests\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_live_checks_total counter\n")
	fmt.Fprintf(w, "asymmetrica_live_checks_total %d\n\n", atomic.LoadInt64(&s.counters.liveChecks))

	// Server uptime gauge
	uptime := time.Since(s.startTime).Seconds()
	fmt.Fprintf(w, "# HELP asymmetrica_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(w, "# TYPE asymmetrica_uptime_seconds gauge\n")
	fmt.Fprintf(w, "asymmetrica_uptime_seconds %f\n", uptime)
}

// writeJSON is a helper to write JSON responses with proper headers
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// parseJSON is a helper to parse JSON request bodies
func parseJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
