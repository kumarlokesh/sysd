// Command wald is a command-line interface for interacting with the Write-Ahead Log.
// It demonstrates the usage of the WAL package with various operations.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/kumarlokesh/sysd/exercises/wal/internal/wal"
)

// Global flags
var (
	helpFlag      = flag.Bool("help", false, "Show help message")
	dir           = flag.String("dir", "./data/wal", "Directory to store WAL segments")
	syncFlag      = flag.Bool("sync", true, "Whether to sync writes to disk")
	segmentSize   = flag.Int64("segment-size", 64*1024*1024, "Maximum size of each segment file in bytes")
	bufferSize    = flag.Int("buffer-size", 64*1024, "Size of the write buffer in bytes")
	flushInterval = flag.Duration("flush-interval", time.Second, "Interval for background flushes")
)

// Command represents a CLI command
type Command struct {
	Name        string
	Description string
	Run         func(config *wal.Config, txMgr *txManager, args []string) error
}

// Available commands
var commands = []Command{
	{
		Name:        "write",
		Description: "Write a key-value pair to the WAL",
		Run:         runWrite,
	},
	{
		Name:        "read",
		Description: "Read all records from the WAL",
		Run:         runRead,
	},
	{
		Name:        "begin-tx",
		Description: "Begin a new transaction",
		Run:         runBeginTx,
	},
	{
		Name:        "commit",
		Description: "Commit a transaction",
		Run:         runCommit,
	},
	{
		Name:        "abort",
		Description: "Abort a transaction",
		Run:         runAbort,
	},
	{
		Name:        "tx-write",
		Description: "Write a key-value pair in a transaction",
		Run:         runTxWrite,
	},
}

func main() {
	// Set up panic recovery for graceful error handling
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			os.Exit(1)
		}
	}()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [command] [arguments]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "\nAvailable commands:\n")
		for _, cmd := range commands {
			fmt.Fprintf(flag.CommandLine.Output(), "  %-10s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Fprintf(flag.CommandLine.Output(), "\nGlobal flags:\n")
		flag.PrintDefaults()

		fmt.Fprintf(flag.CommandLine.Output(), "\nExamples:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  Write a key-value pair:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  \t%s write -key mykey -value myvalue\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  \n  Read all records:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  \t%s read\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  \n  Use a transaction:\n")
		fmt.Fprintf(flag.CommandLine.Output(), "  \t%s begin-tx\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  \t%s tx-write -key txkey -value txvalue\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "  \t%s commit\n", os.Args[0])
	}

	flag.Parse()

	if *helpFlag || len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	// Get the subcommand
	var cmd *Command
	for i := range commands {
		if commands[i].Name == flag.Arg(0) {
			cmd = &commands[i]
			break
		}
	}

	if cmd == nil {
		flag.Usage()
		os.Exit(1)
	}

	// Create WAL directory if it doesn't exist
	abspath, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	if err := os.MkdirAll(abspath, 0755); err != nil {
		log.Fatalf("Failed to create WAL directory: %v", err)
	}

	// Initialize transaction manager
	txMgr := NewTxManager(abspath).(*txManager)

	// Create WAL configuration
	config := &wal.Config{
		Dir:           abspath,
		Sync:          *syncFlag,
		SegmentSize:   *segmentSize,
		BufferSize:    *bufferSize,
		FlushInterval: *flushInterval,
	}

	// Run the command
	if err := cmd.Run(config, txMgr, flag.Args()[1:]); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// Command implementations

func runWrite(config *wal.Config, txMgr *txManager, args []string) error {
	fs := flag.NewFlagSet("write", flag.ExitOnError)
	key := fs.String("key", "", "Key to write")
	value := fs.String("value", "", "Value to write")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if *key == "" || *value == "" {
		return fmt.Errorf("both key and value must be specified")
	}

	// Open WAL
	w, err := wal.Open(config)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer w.Close()

	// Write the record (non-transactional)
	lsn, err := w.Write(0, []byte(*key), []byte(*value))
	if err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	fmt.Printf("Wrote record: LSN=%d, key=%s, value=%s\n", lsn, *key, *value)
	return nil
}

func runRead(config *wal.Config, txMgr *txManager, args []string) error {
	// Open WAL
	w, err := wal.Open(config)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer w.Close()

	// Read all records
	records, err := w.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read from WAL: %w", err)
	}

	fmt.Println("Records in WAL:")
	fmt.Println("LSN      | TxID  | Type  | Key              | Value")
	fmt.Println("---------|-------|-------|-----------------|-----------------")
	for _, r := range records {
		fmt.Printf("%-8d | %-5d | %-5d | %-15s | %s\n",
			r.LSN, r.TxID, r.Type, string(r.Key), string(r.Value))
	}

	return nil
}

func runBeginTx(config *wal.Config, txMgr *txManager, args []string) error {
	// Begin a new transaction
	txID, err := txMgr.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Open WAL to verify the transaction can be started
	w, err := wal.Open(config)
	if err != nil {
		// If we can't open the WAL, clean up the transaction
		if err := txMgr.End(txID, false); err != nil {
			return fmt.Errorf("failed to end transaction: %w", err)
		}
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	w.Close()

	fmt.Printf("Started new transaction %d\n", txID)
	return nil
}

func runCommit(config *wal.Config, txMgr *txManager, args []string) error {
	// Get the active transaction
	txID, active, err := txMgr.GetActiveTx()
	if err != nil {
		return fmt.Errorf("failed to get active transaction: %w", err)
	}

	if !active {
		return fmt.Errorf("no active transaction to commit")
	}

	// Open WAL
	w, err := wal.Open(config)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer w.Close()

	// Commit the transaction
	if err := w.Commit(txID); err != nil {
		return fmt.Errorf("failed to commit transaction %d: %w", txID, err)
	}

	// Mark transaction as committed in the manager
	if err := txMgr.End(txID, true); err != nil {
		return fmt.Errorf("failed to end transaction %d: %w", txID, err)
	}

	fmt.Printf("Committed transaction %d\n", txID)
	return nil
}

func runAbort(config *wal.Config, txMgr *txManager, args []string) error {
	// Get the active transaction
	txID, active, err := txMgr.GetActiveTx()
	if err != nil {
		return fmt.Errorf("failed to get active transaction: %w", err)
	}

	if !active {
		return fmt.Errorf("no active transaction to abort")
	}

	// Open WAL
	w, err := wal.Open(config)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer w.Close()

	// Abort the transaction
	if err := w.Abort(txID); err != nil {
		return fmt.Errorf("failed to abort transaction %d: %w", txID, err)
	}

	// Mark transaction as aborted in the manager
	if err := txMgr.End(txID, false); err != nil {
		return fmt.Errorf("failed to end transaction %d: %w", txID, err)
	}

	fmt.Printf("Aborted transaction %d\n", txID)
	return nil
}

func runTxWrite(config *wal.Config, txMgr *txManager, args []string) error {
	// Get the active transaction
	txID, active, err := txMgr.GetActiveTx()
	if err != nil {
		return fmt.Errorf("failed to get active transaction: %w", err)
	}

	if !active {
		return fmt.Errorf("no active transaction - use 'begin-tx' first")
	}

	// Parse command line flags
	fs := flag.NewFlagSet("tx-write", flag.ExitOnError)
	key := fs.String("key", "", "Key to write")
	value := fs.String("value", "", "Value to write")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("failed to parse flags: %w", err)
	}

	if *key == "" || *value == "" {
		return fmt.Errorf("both key and value must be specified")
	}

	// Open WAL
	w, err := wal.Open(config)
	if err != nil {
		return fmt.Errorf("failed to open WAL: %w", err)
	}
	defer w.Close()

	// Write the record in the transaction
	lsn, err := w.Write(txID, []byte(*key), []byte(*value))
	if err != nil {
		return fmt.Errorf("failed to write to WAL: %w", err)
	}

	// Ensure the write is flushed to disk
	if err := w.Sync(); err != nil {
		return fmt.Errorf("failed to sync WAL: %w", err)
	}

	fmt.Printf("Wrote record: LSN=%d, TxID=%d, key=%s, value=%s\n", lsn, txID, *key, *value)
	return nil
}
