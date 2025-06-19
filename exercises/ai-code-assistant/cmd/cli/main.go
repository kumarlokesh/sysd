package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kumarlokesh/sysd/exercises/ai-code-assistant/internal/config"
)

func main() {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
		},
		ChromaDB: config.ChromaDBConfig{
			URL:    "http://localhost:8000",
			APIKey: "",
		},
		LLM: config.LLMConfig{
			Model:       "codellama:7b",
			Temperature: 0.7,
			MaxTokens:   2000,
			Timeout:     30 * time.Second,
		},
	}

	help := flag.Bool("help", false, "Show help message")
	version := flag.Bool("version", false, "Show version information")

	flag.Parse()

	if *help {
		showHelp()
		os.Exit(0)
	}
	if *version {
		showVersion()
		os.Exit(0)
	}
	if flag.NArg() == 0 {
		showHelp()
		os.Exit(1)
	}

	args := flag.Args()
	subcommand := args[0]
	subcommandArgs := args[1:]
	switch subcommand {
	case "config":
		handleConfigCommand(cfg, subcommandArgs)
	case "index":
		handleIndexCommand(cfg, subcommandArgs)
	case "query":
		handleQueryCommand(cfg, subcommandArgs)
	case "chat":
		handleChatCommand(cfg, subcommandArgs)
	default:
		log.Printf("Unknown command: %s\n\n", subcommand)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	helpText := `AI Code Assistant CLI

Usage:
  ai-code-assistant [flags] <command> [arguments]

Flags:
  --config string   Path to config file
  --help            Show this help message
  --version         Show version information

Commands:
  config           Show current configuration
  index <path>     Index a directory or file
  query <text>     Query the codebase
  chat             Start interactive chat mode
`
	fmt.Print(helpText)
}

func showVersion() {
	fmt.Println("AI Code Assistant v0.1.0")
}

func handleConfigCommand(cfg *config.Config, args []string) {
	fmt.Println("Current configuration:")
	fmt.Printf("Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("ChromaDB URL: %s\n", cfg.ChromaDB.URL)
	if cfg.ChromaDB.APIKey != "" {
		fmt.Println("ChromaDB API Key: [set]")
	} else {
		fmt.Println("ChromaDB API Key: [not set]")
	}
	fmt.Printf("LLM Model: %s\n", cfg.LLM.Model)
}

func handleIndexCommand(cfg *config.Config, args []string) {
	if len(args) == 0 {
		log.Fatal("Please provide a directory or file to index")
	}

	path := args[0]
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		log.Fatalf("Path does not exist: %s", absPath)
	}

	// TODO: Implement actual indexing
	if info.IsDir() {
		fmt.Printf("Indexing directory: %s\n", absPath)
	} else {
		fmt.Printf("Indexing file: %s\n", absPath)
	}

	// TODO: Connect to database and store embeddings
	fmt.Println("Indexing complete!")
}

func handleQueryCommand(cfg *config.Config, args []string) {
	if len(args) == 0 {
		log.Fatal("Please provide a query")
	}

	query := strings.Join(args, " ")
	// TODO: Implement query processing
	fmt.Printf("Query: %s\n", query)
	fmt.Println("This would search the indexed codebase for relevant code snippets.")
}

func handleChatCommand(cfg *config.Config, args []string) {
	// TODO: Implement interactive chat mode
	fmt.Println("Starting interactive chat mode (not implemented yet)")
	fmt.Println("Type 'exit' or 'quit' to end the session.")
	fmt.Println()

	// Simple read-eval-print loop (REPL)
	for {
		fmt.Print("\nYou: ")
		var input string
		_, err := fmt.Scanln(&input)
		if err != nil {
			log.Printf("Error reading input: %v", err)
			continue
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			break
		}

		// TODO: Process the input and generate a response
		fmt.Println("AI: I'm a simple AI assistant. This feature is not fully implemented yet.")
		fmt.Printf("You said: %s\n", input)
	}
}
