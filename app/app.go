package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/swissinfo-ch/zoe/ev"
	"github.com/swissinfo-ch/zoe/report"
	"golang.org/x/time/rate"
)

type App struct {
	ctx            context.Context
	laddr          string
	clients        map[string]*client // writer:addr or reader:addr
	clientMu       sync.Mutex
	events         chan *ev.Ev
	filename       string
	reportRunner   *report.Runner
	reportNames    []string
	commit         string
	blockSize      int
	rateLimitEvery time.Duration
	rateLimitBurst int
	numCPU         int
}

type AppCfg struct {
	Ctx            context.Context
	Laddr          string
	Filename       string
	BlockSize      int
	ReportRunner   *report.Runner
	ReportNames    []string
	RateLimitEvery time.Duration
	RateLimitBurst int
}

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewApp creates & starts a new App.
func NewApp(cfg *AppCfg) *App {
	a := &App{
		ctx:            cfg.Ctx,
		filename:       cfg.Filename,
		clients:        make(map[string]*client),
		clientMu:       sync.Mutex{},
		events:         make(chan *ev.Ev, 100),
		reportRunner:   cfg.ReportRunner,
		reportNames:    cfg.ReportNames,
		blockSize:      cfg.BlockSize,
		rateLimitEvery: cfg.RateLimitEvery,
		rateLimitBurst: cfg.RateLimitBurst,
		numCPU:         runtime.NumCPU(),
	}
	commit, err := os.ReadFile("commit")
	if err != nil {
		panic(fmt.Sprintf("failed to read commit file: %v", err))
	}
	a.commit = string(commit)
	go a.cleanupVisitors()
	go a.writeEvents()
	go a.serve()
	return a
}

// serve starts the HTTP server.
func (a *App) serve() {
	mux := http.NewServeMux()
	mux.Handle("/", a.rateLimitMiddleware(
		a.corsMiddleware(
			http.HandlerFunc(a.handleRequest))))
	fmt.Println("app listening http on", a.laddr)
	server := &http.Server{Addr: a.laddr, Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			panic(fmt.Sprintf("failed to listen http: %v\n", err))
		}
	}()
	<-a.ctx.Done()
	a.shutdown(server)
}

// handleRequest is the HTTP handler for all endpoints.
func (a *App) handleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		switch r.URL.Path {
		case "/":
			a.handleRoot(w, r)
		case "/r":
			a.handleGetReportResult(w, r)
		case "/js":
			a.handleGetJS(w, r)
		case "/status":
			a.handleGetStatus(w, r)
		default:
			http.ServeFile(w, r, "assets"+r.URL.Path)
		}
	case "POST":
		a.handlePost(w, r)
	case "OPTIONS":
		w.WriteHeader(http.StatusOK)
	}
}

func (a *App) handleGetJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	w.Header().Set("Cache-Control", "max-age=3600")
	http.ServeFile(w, r, "assets/client.js")
}

// rateLimitMiddleware is a middleware that limits the rate of requests.
func (a *App) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limiter := a.getRateLimiter(r)
		if !limiter.Allow() &&
			// TODO: REMOVE. TEST ONLY! Disables rate limit for POST
			r.Method != "POST" {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO: check origin header is swissinfo.ch
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "X_TYPE,X_USR,X_SESS,X_CID,X_SCROLLED,X_PAGE_SECONDS")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		next.ServeHTTP(w, r)
	})
}

// getRateLimiter returns a rate limiter for the given request.
func (a *App) getRateLimiter(r *http.Request) *rate.Limiter {
	addr := r.Header.Get("Fly-Client-IP")
	if addr == "" {
		addr = r.RemoteAddr // fallback when local
	}
	a.clientMu.Lock()
	defer a.clientMu.Unlock()
	key := r.Method + addr
	v, exists := a.clients[key]
	if !exists {
		limiter := rate.NewLimiter(rate.Every(a.rateLimitEvery), a.rateLimitBurst)
		a.clients[key] = &client{limiter, time.Now()}
		return limiter
	}
	v.lastSeen = time.Now()
	return v.limiter
}

// cleanupVisitors removes clients that have not been seen for 10 seconds.
func (a *App) cleanupVisitors() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-time.After(10 * time.Second):
			a.clientMu.Lock()
			for key, client := range a.clients {
				if time.Since(client.lastSeen) > 10*time.Second {
					delete(a.clients, key)
				}
			}
			a.clientMu.Unlock()
		}
	}
}

// shutdown attempts to gracefully shutdown the server.
func (a *App) shutdown(server *http.Server) {
	// Create a context with timeout for the server shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Attempt to gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		panic(fmt.Sprintf("server shutdown failed: %v", err))
	}
	fmt.Println("server shutdown gracefully")
}
