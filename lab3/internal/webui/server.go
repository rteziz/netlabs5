// Package webui serves the local UI and non-blocking asynchronous job endpoints.
package webui

import (
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"math"
	"net/http"
	"places/internal/future"
	"places/internal/places"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	searchTimeout   = 20 * time.Second
	exploreTimeout  = 35 * time.Second
	jobRetention    = 2 * time.Minute
	maxJobs         = 64
	maxRequestBytes = 4096
	maxQueryLength  = 150
	minRadius       = 10
	maxRadius       = 10000
	minLimit        = 1
	maxLimit        = 10
	tokenBytes      = 16
)

//go:embed static/*
var assets embed.FS

type Options struct {
	Language  string `json:"language"`
	PlaceType string `json:"placeType"`
	Radius    int    `json:"radius"`
	Limit     int    `json:"limit"`
}
type request struct {
	Options
	Query    string          `json:"query"`
	Location places.Location `json:"location"`
}
type Service interface {
	SearchLocations(context.Context, string) *future.Future[[]places.Location]
	Explore(context.Context, places.Location) *future.Future[places.Result]
}
type Factory func(string, int, int, string) Service

type job struct {
	Status  string `json:"status"`
	Result  any    `json:"result,omitempty"`
	Error   string `json:"error,omitempty"`
	cancel  context.CancelFunc
	expires time.Time
}
type Server struct {
	ctx      context.Context
	defaults Options
	factory  Factory
	mu       sync.Mutex
	jobs     map[string]*job
}

func New(ctx context.Context, defaults Options, factory Factory) *Server {
	if factory == nil {
		factory = func(lang string, radius, limit int, placeType string) Service {
			s := places.New(lang, radius, limit)
			s.PlaceType = placeType
			return s
		}
	}
	return &Server{ctx: ctx, defaults: defaults, factory: factory, jobs: make(map[string]*job)}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}
		writeJSON(w, http.StatusOK, s.defaults)
	})
	mux.HandleFunc("/api/place-types", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, http.StatusOK, places.PlaceTypes) })
	mux.HandleFunc("/api/search", s.start)
	mux.HandleFunc("/api/explore", s.start)
	mux.HandleFunc("/api/jobs/", s.poll)
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) { fail(w, http.StatusNotFound, "not_found") })
	files, _ := fs.Sub(assets, "static")
	mux.Handle("/", http.FileServer(http.FS(files)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data: https://upload.wikimedia.org; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
}
func valid(o Options) bool {
	return places.ValidPlaceType(o.PlaceType) && (o.Language == "ru" || o.Language == "en") && o.Radius >= minRadius && o.Radius <= maxRadius && o.Limit >= minLimit && o.Limit <= maxLimit
}
func (s *Server) start(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	if strings.Split(r.Header.Get("Content-Type"), ";")[0] != "application/json" {
		fail(w, http.StatusUnsupportedMediaType, "invalid_request")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	var input request
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	input.PlaceType = strings.ToLower(strings.TrimSpace(input.PlaceType))
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF || !valid(input.Options) {
		fail(w, http.StatusBadRequest, "invalid_request")
		return
	}
	search := r.URL.Path == "/api/search"
	input.Query = strings.TrimSpace(input.Query)
	if search && (input.Query == "" || utf8.RuneCountInString(input.Query) > maxQueryLength) {
		fail(w, http.StatusBadRequest, "invalid_query")
		return
	}
	loc := input.Location
	if !search && (strings.TrimSpace(loc.Name) == "" || len(loc.Name) > maxRequestBytes/2 || math.IsNaN(loc.Latitude) || math.IsNaN(loc.Longitude) || math.Abs(loc.Latitude) > 90 || math.Abs(loc.Longitude) > 180) {
		fail(w, http.StatusBadRequest, "invalid_location")
		return
	}
	timeout := exploreTimeout
	if search {
		timeout = searchTimeout
	}
	id, ctx, err := s.create(timeout)
	if err != nil {
		fail(w, http.StatusServiceUnavailable, "busy")
		return
	}
	service := s.factory(input.Language, input.Radius, input.Limit, input.PlaceType)
	stop := context.AfterFunc(ctx, func() { s.finish(id, nil, ctx.Err()) })
	complete := func(value any, err error) { stop(); s.finish(id, value, err) }
	if search {
		service.SearchLocations(ctx, input.Query).OnComplete(func(value []places.Location, err error) { complete(value, err) })
	} else {
		service.Explore(ctx, loc).OnComplete(func(value places.Result, err error) { complete(value, err) })
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"id": id})
}
func (s *Server) create(timeout time.Duration) (string, context.Context, error) {
	var token [tokenBytes]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", nil, err
	}
	id := hex.EncodeToString(token[:])
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	for key, value := range s.jobs {
		if now.After(value.expires) {
			value.cancel()
			delete(s.jobs, key)
		}
	}
	if len(s.jobs) >= maxJobs {
		return "", nil, errors.New("job limit reached")
	}
	ctx, cancel := context.WithTimeout(s.ctx, timeout)
	s.jobs[id] = &job{Status: "pending", cancel: cancel, expires: now.Add(jobRetention)}
	return id, ctx, nil
}
func (s *Server) finish(id string, value any, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	j, ok := s.jobs[id]
	if !ok || j.Status != "pending" {
		return
	}
	if err != nil {
		j.Status = "error"
		j.Error = "upstream"
		if errors.Is(err, places.ErrMissingAPIKey) {
			j.Error = "missing_key"
		} else if strings.Contains(err.Error(), "HTTP 401") || strings.Contains(err.Error(), "HTTP 403") {
			j.Error = "auth"
		} else if errors.Is(err, context.DeadlineExceeded) {
			j.Error = "timeout"
		} else if errors.Is(err, context.Canceled) {
			j.Error = "canceled"
		} else if strings.Contains(err.Error(), "HTTP 429") {
			j.Error = "rate_limit"
		}
	} else {
		j.Status = "done"
		j.Result = value
	}
	j.cancel()
}
func (s *Server) poll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodDelete {
		fail(w, http.StatusMethodNotAllowed, "method_not_allowed")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	s.mu.Lock()
	j, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		fail(w, http.StatusNotFound, "expired")
		return
	}
	if time.Now().After(j.expires) {
		j.cancel()
		delete(s.jobs, id)
		s.mu.Unlock()
		fail(w, http.StatusNotFound, "expired")
		return
	}
	if r.Method == http.MethodDelete {
		j.cancel()
		delete(s.jobs, id)
		s.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	snapshot := *j
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, snapshot)
}
func fail(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
