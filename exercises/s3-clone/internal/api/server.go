package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kumarlokesh/s3-clone/internal/storage"
	"github.com/kumarlokesh/s3-clone/internal/types"
)

// Server represents the HTTP API server
type Server struct {
	storage storage.Storage
	server  *http.Server
	addr    string
	cancel  context.CancelFunc
	ctx     context.Context
}

// NewServer creates a new API server
func NewServer(addr string, store storage.Storage) *Server {
	s := &Server{
		storage: store,
		addr:    addr,
	}

	r := mux.NewRouter()

	// Add request logging middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("Received request: %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// List all buckets
	r.HandleFunc("/", s.listBuckets).Methods("GET")

	// Bucket operations
	r.HandleFunc("/{bucket}", s.createBucket).Methods("PUT")
	r.HandleFunc("/{bucket}", s.deleteBucket).Methods("DELETE")
	r.HandleFunc("/{bucket}", s.listObjects).Methods("GET")

	// Object operations
	r.HandleFunc("/{bucket}/{key:.+}", s.putObject).Methods("PUT")
	r.HandleFunc("/{bucket}/{key:.+}", s.getObject).Methods("GET")
	r.HandleFunc("/{bucket}/{key:.+}", s.deleteObject).Methods("DELETE")

	// Add a catch-all route for debugging
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("No route matched: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	})

	// Ensure the address includes a host if not specified
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = "0.0.0.0:" + addr
	}

	s.server = &http.Server{
		Addr:    addr,
		Handler: r,
	}

	return s
}

// Handler returns the HTTP handler for the server
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

// Addr returns the address the server is configured to listen on
func (s *Server) Addr() string {
	return s.addr
}

// Start starts the HTTP server and blocks until the server is shut down
func (s *Server) Start() error {
	addr := s.server.Addr
	if addr == "" {
		addr = ":http"
	}

	// Ensure the address is in host:port format
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// If splitting fails, assume it's just a port
		host = ""
		port = addr
	}

	// If host is empty, bind to all interfaces
	if host == "" {
		host = "0.0.0.0" // Bind to all interfaces
	}

	// Reconstruct the address
	addr = net.JoinHostPort(host, port)
	s.server.Addr = addr

	log.Printf("Configuring server to listen on %s", addr)

	// Create a listener first to catch any errors early
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to create listener on %s: %v", addr, err)
		return fmt.Errorf("failed to listen on %s: %v", addr, err)
	}

	// Get the actual address we're listening on (in case port 0 was used)
	actualAddr := listener.Addr().String()
	log.Printf("Server listening on %s", actualAddr)

	s.ctx, s.cancel = context.WithCancel(context.Background())

	// Channel to signal when the server has started
	started := make(chan struct{})

	// Start serving in a goroutine
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Starting to serve HTTP requests on %s", actualAddr)
		// Signal that we've started serving
		close(started)
		// This will block until the server is shut down
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %v", err)
		}
	}()

	// Verify the server is running
	select {
	case <-started:
		// Server has started, verify it's accepting connections
		conn, err := net.DialTimeout("tcp", actualAddr, 100*time.Millisecond)
		if err != nil {
			return fmt.Errorf("server is not accepting connections: %v", err)
		}
		conn.Close()
		log.Printf("Server successfully started and accepting connections on %s", actualAddr)

		// Block until the context is cancelled
		select {
		case <-s.ctx.Done():
			log.Println("Server context cancelled, shutting down...")
			return nil
		case err := <-errChan:
			return err
		}

	case err := <-errChan:
		return err

	case <-time.After(2 * time.Second):
		return fmt.Errorf("timeout waiting for server to start on %s", actualAddr)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")
	if s.cancel != nil {
		s.cancel()
	}
	return s.server.Shutdown(ctx)
}

// Helper functions for HTTP responses
func (s *Server) respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, err error) {
	s.respond(w, status, map[string]string{"error": err.Error()})
}

// HTTP Handlers
// listBuckets handles GET / - List all buckets
func (s *Server) listBuckets(w http.ResponseWriter, r *http.Request) {
	buckets, err := s.storage.ListBuckets(r.Context())
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]interface{}{
		"buckets": buckets,
	})
}

func (s *Server) createBucket(w http.ResponseWriter, r *http.Request) {
	bucket := mux.Vars(r)["bucket"]
	if err := s.storage.CreateBucket(r.Context(), bucket); err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}
	s.respond(w, http.StatusOK, map[string]string{"message": "bucket created"})
}

func (s *Server) deleteBucket(w http.ResponseWriter, r *http.Request) {
	bucket := mux.Vars(r)["bucket"]
	if err := s.storage.DeleteBucket(r.Context(), bucket); err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}
	s.respond(w, http.StatusNoContent, nil)
}

func (s *Server) listObjects(w http.ResponseWriter, r *http.Request) {
	bucket := mux.Vars(r)["bucket"]
	prefix := r.URL.Query().Get("prefix")

	objects, err := s.storage.ListObjects(r.Context(), bucket, prefix)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}

	// Convert to a simpler format for the response
	var keys []string
	for _, obj := range objects {
		keys = append(keys, obj.Key)
	}

	s.respond(w, http.StatusOK, map[string]interface{}{
		"bucket":  bucket,
		"prefix":  prefix,
		"objects": keys,
	})
}

func (s *Server) putObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	data, err := io.ReadAll(r.Body)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, fmt.Errorf("failed to read request body: %v", err))
		return
	}

	opts := &types.PutObjectOptions{
		ContentType: r.Header.Get("Content-Type"),
		Metadata:    make(map[string]string),
	}

	// Copy user-defined metadata
	for k, v := range r.Header {
		if strings.HasPrefix(k, "X-Amz-Meta-") {
			metaKey := strings.TrimPrefix(k, "X-Amz-Meta-")
			if len(v) > 0 {
				opts.Metadata[metaKey] = v[0]
			}
		}
	}

	if err := s.storage.PutObject(r.Context(), bucket, key, data, opts); err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}

	s.respond(w, http.StatusOK, map[string]string{
		"bucket": bucket,
		"key":    key,
	})
}

func (s *Server) getObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	obj, err := s.storage.GetObject(r.Context(), bucket, key, &types.GetObjectOptions{})
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}
	if obj == nil {
		s.respondError(w, http.StatusNotFound, fmt.Errorf("object not found"))
		return
	}

	w.Header().Set("Content-Type", obj.ContentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", obj.Size))
	w.Header().Set("Last-Modified", obj.ModifiedAt.UTC().Format(http.TimeFormat))
	for k, v := range obj.Metadata {
		w.Header().Set("X-Amz-Meta-"+k, v)
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(obj.Content)
}

func (s *Server) deleteObject(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bucket := vars["bucket"]
	key := vars["key"]

	if err := s.storage.DeleteObject(r.Context(), bucket, key); err != nil {
		s.respondError(w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
