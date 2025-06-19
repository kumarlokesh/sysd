package indexer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
)

// Parser is responsible for parsing code files into syntax trees
type Parser struct {
	parser *sitter.Parser
	mutex  sync.Mutex
}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	log.Debug().Msg("Creating new tree-sitter parser")

	// Create the parser
	parser := sitter.NewParser()
	if parser == nil {
		panic("failed to create tree-sitter parser")
	}

	log.Debug().Msg("Successfully created tree-sitter parser")
	return &Parser{
		parser: parser,
	}
}

// getLanguageConfig returns the tree-sitter language configuration for the given language name
func getLanguageConfig(language string) (*sitter.Language, error) {
	log.Debug().Str("requested_language", language).Msg("Getting language configuration")

	switch strings.ToLower(language) {
	case "go":
		log.Debug().Msg("Loading Go language configuration")
		lang := golang.GetLanguage()
		if lang == nil {
			return nil, fmt.Errorf("failed to load Go language configuration")
		}
		return lang, nil
	case "python":
		log.Debug().Msg("Loading Python language configuration")
		lang := python.GetLanguage()
		if lang == nil {
			return nil, fmt.Errorf("failed to load Python language configuration")
		}
		return lang, nil
	case "javascript", "typescript":
		log.Debug().Msg("Loading JavaScript language configuration")
		lang := javascript.GetLanguage()
		if lang == nil {
			return nil, fmt.Errorf("failed to load JavaScript language configuration")
		}
		return lang, nil
	default:
		err := fmt.Errorf("unsupported language: %s", language)
		log.Error().Err(err).Str("language", language).Msg("Unsupported language")
		return nil, err
	}
}

// Parse parses the given source code into a syntax tree
func (p *Parser) Parse(content []byte, language string) (*sitter.Tree, error) {
	if p == nil || p.parser == nil {
		return nil, errors.New("parser is not initialized")
	}

	if len(content) == 0 {
		return nil, errors.New("empty content provided for parsing")
	}

	log.Debug().
		Str("language", language).
		Int("content_length", len(content)).
		Msg("Starting to parse content")

	lang, err := getLanguageConfig(language)
	if err != nil {
		log.Error().Err(err).Str("language", language).Msg("Failed to get language config")
		return nil, fmt.Errorf("failed to get language config: %w", err)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	log.Debug().
		Str("language", language).
		Msg("Setting language on parser")

	p.parser.SetLanguage(lang)

	ctx := context.Background()

	log.Debug().Msg("Starting to parse content with tree-sitter")
	tree, err := p.parser.ParseCtx(ctx, nil, content)
	if err != nil {
		log.Error().
			Err(err).
			Str("language", language).
			Int("content_length", len(content)).
			Msg("Failed to parse content")
		return nil, fmt.Errorf("failed to parse content: %w", err)
	}

	if tree == nil {
		err := errors.New("parsing resulted in a nil tree")
		log.Error().
			Err(err).
			Str("language", language).
			Msg("Parsing resulted in nil tree")
		return nil, err
	}

	log.Debug().
		Str("language", language).
		Int("content_length", len(content)).
		Msg("Successfully parsed content")

	return tree, nil
}

// Close releases resources used by the parser
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

// GetNodeContent returns the source code content for a node
func GetNodeContent(content []byte, node *sitter.Node) string {
	return string(content[node.StartByte():node.EndByte()])
}

// GetNodePosition returns the start and end line numbers of a node
func GetNodePosition(node *sitter.Node) (startLine, endLine int) {
	startPoint := node.StartPoint()
	endPoint := node.EndPoint()
	return int(startPoint.Row) + 1, int(endPoint.Row) + 1 // Convert to 1-based line numbers
}

// GetNodeLines returns the lines of code for a node
func GetNodeLines(content []byte, node *sitter.Node) []string {
	nodeContent := GetNodeContent(content, node)
	return strings.Split(nodeContent, "\n")
}

// GetLeadingWhitespace returns the leading whitespace of a line
func GetLeadingWhitespace(line string) string {
	return line[:len(line)-len(strings.TrimLeft(line, " \t"))]
}

// FormatNode formats a node with proper indentation
func FormatNode(node *sitter.Node, content []byte) string {
	nodeContent := GetNodeContent(content, node)
	lines := strings.Split(nodeContent, "\n")

	if len(lines) == 0 {
		return ""
	}

	// Find the minimum indentation level (excluding empty lines)
	minIndent := ""
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := GetLeadingWhitespace(line)
		if minIndent == "" || len(indent) < len(minIndent) {
			minIndent = indent
		}
	}

	// Remove the minimum indentation from all lines
	var result bytes.Buffer
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			// Remove the minimum indentation, but keep relative indentation
			line = strings.TrimPrefix(line, minIndent)
		}
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
