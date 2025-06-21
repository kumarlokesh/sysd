package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/kumarlokesh/s3-clone/internal/api"
	"github.com/kumarlokesh/s3-clone/internal/metadata"
	"github.com/kumarlokesh/s3-clone/internal/storage"
)

const (
	defaultAddr       = ":8080"
	defaultStorageDir = "./data"
)

// StorageType represents the type of storage backend
type StorageType string

const (
	// StorageTypeMemory is the in-memory storage backend
	StorageTypeMemory StorageType = "memory"
	// StorageTypeFilesystem is the filesystem-based storage backend
	StorageTypeFilesystem StorageType = "filesystem"
)

func main() {
	addr := flag.String("addr", defaultAddr, "server address")
	storageType := flag.String("storage", string(StorageTypeMemory), "storage backend (memory or filesystem)")
	dataDir := flag.String("data-dir", defaultStorageDir, "data directory for filesystem storage")
	enableDebug := flag.Bool("debug", false, "enable debug logging")
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if *enableDebug {
		log.Println("Debug logging enabled")
	}

	// Initialize metadata service
	metaSvc := metadata.NewInMemoryMetadata()

	// Initialize storage based on type
	var store storage.Storage

	switch StorageType(*storageType) {
	case StorageTypeMemory:
		log.Println("Using in-memory storage")
		store = storage.NewMemoryStorage(metaSvc)

	case StorageTypeFilesystem:
		// Ensure data directory exists
		absDataDir, err := filepath.Abs(*dataDir)
		if err != nil {
			log.Fatalf("Invalid data directory: %v", err)
		}

		if *enableDebug {
			log.Printf("Using filesystem storage at: %s", absDataDir)
		}

		store, err = storage.NewFilesystemStorage(absDataDir, metaSvc)
		if err != nil {
			log.Fatalf("Failed to initialize filesystem storage: %v", err)
		}

	default:
		log.Fatalf("Unsupported storage type: %s", *storageType)
	}

	// Verify storage is working
	if err := store.Ping(context.Background()); err != nil {
		log.Fatalf("Storage ping failed: %v", err)
	}

	// Create the server
	server := api.NewServer(*addr, store)

	// Channel to listen for errors coming from the server
	serverErrors := make(chan error, 1)

	// Start the server in a goroutine
	serverStarted := make(chan bool, 1)
	go func() {
		log.Printf("Starting S3 clone server on %s", *addr)
		serverStarted <- true
		if err := server.Start(); err != nil {
			serverErrors <- err
		}
	}()

	// Channel to listen for interrupt or terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	log.Println("Waiting for server to start...")

	// Wait for the server to start or an error to occur
	select {
	case <-serverStarted:
		log.Printf("Server is running and accepting connections at http://localhost%s", *addr)
		log.Printf("Press Ctrl+C to stop the server")

		// Block until we receive a signal or an error from the server
		select {
		case err := <-serverErrors:
			log.Fatalf("Server error: %v", err)
		case sig := <-stop:
			log.Printf("Received signal %v, shutting down server...", sig)
		}

	case err := <-serverErrors:
		log.Fatalf("Failed to start server: %v", err)

	case sig := <-stop:
		log.Printf("Received signal %v before server started, shutting down...", sig)
		return
	}

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	log.Println("Initiating graceful shutdown...")
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Error during server shutdown: %v", err)
	} else {
		log.Println("Server gracefully stopped")
	}

	log.Println("Server stopped")
}
