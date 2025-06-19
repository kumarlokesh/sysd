package indexer

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// Chunker is responsible for splitting code into meaningful chunks
type Chunker struct {
	// Minimum and maximum chunk sizes in characters
	minChunkSize int
	maxChunkSize int

	// Whether to split functions into smaller chunks if they exceed maxChunkSize
	splitLargeFunctions bool
}

// NewChunker creates a new Chunker with default settings
func NewChunker() *Chunker {
	return &Chunker{
		minChunkSize:        100,  // Minimum 100 characters per chunk
		maxChunkSize:        2000, // Maximum 2000 characters per chunk
		splitLargeFunctions: true, // Split large functions into smaller chunks
	}
}

// WithMinChunkSize sets the minimum chunk size
func (c *Chunker) WithMinChunkSize(size int) *Chunker {
	c.minChunkSize = size
	return c
}

// WithMaxChunkSize sets the maximum chunk size
func (c *Chunker) WithMaxChunkSize(size int) *Chunker {
	c.maxChunkSize = size
	return c
}

// WithSplitLargeFunctions sets whether to split large functions
func (c *Chunker) WithSplitLargeFunctions(split bool) *Chunker {
	c.splitLargeFunctions = split
	return c
}

// ChunkFile chunks a file into smaller pieces
func (c *Chunker) ChunkFile(filePath string, content []byte, language string, tree *sitter.Tree) ([]Chunk, error) {
	// Get the base file name for chunk metadata
	fileName := filepath.Base(filePath)

	if len(content) == 0 {
		return nil, nil
	}

	if language == "" {
		return []Chunk{{
			ID:        generateChunkID(filePath, 0, 0, 0, 0),
			Content:   string(content),
			FilePath:  filePath,
			Language:  "",
			StartLine: 1,
			EndLine:   bytesCountToLines(content),
			NodeType:  "file",
			Metadata: map[string]string{
				"file_name": fileName,
			},
		}}, nil
	}

	rootNode := tree.RootNode()

	var chunks []Chunk
	switch strings.ToLower(language) {
	case "go":
		chunks = c.chunkGo(rootNode, content, filePath, language)
	case "python":
		chunks = c.chunkPython(rootNode, content, filePath, language)
	case "javascript", "typescript":
		chunks = c.chunkJavaScript(rootNode, content, filePath, language)
	default:
		chunks = c.chunkGeneric(rootNode, content, filePath, language)
	}

	return c.processChunks(chunks, content), nil
}

func (c *Chunker) processChunks(chunks []Chunk, content []byte) []Chunk {
	var result []Chunk

	for _, chunk := range chunks {
		// If chunk is too small, try to merge with adjacent chunks
		if len(chunk.Content) < c.minChunkSize && len(result) > 0 {
			lastChunk := &result[len(result)-1]
			if len(lastChunk.Content)+len(chunk.Content) <= c.maxChunkSize {
				// Merge with previous chunk
				lastChunk.Content = lastChunk.Content + "\n\n" + chunk.Content
				lastChunk.EndLine = chunk.EndLine
				continue
			}
		}

		// If chunk is too large, split it
		if len(chunk.Content) > c.maxChunkSize && c.splitLargeFunctions {
			splitChunks := c.splitLargeChunk(chunk, content)
			result = append(result, splitChunks...)
		} else {
			result = append(result, chunk)
		}
	}

	return result
}

// splitLargeChunk splits a chunk that exceeds the maximum size
func (c *Chunker) splitLargeChunk(chunk Chunk, content []byte) []Chunk {
	// For now, just split by lines and create new chunks
	// A more sophisticated implementation might want to split at logical boundaries
	lines := strings.Split(chunk.Content, "\n")
	if len(lines) <= 1 {
		return []Chunk{chunk}
	}

	var chunks []Chunk
	var currentChunk strings.Builder
	currentStartLine := chunk.StartLine
	lineCount := 0

	for i, line := range lines {
		currentChunk.WriteString(line)
		lineCount++

		// If we've reached the max chunk size or this is the last line
		if currentChunk.Len() >= c.maxChunkSize || i == len(lines)-1 {
			chunks = append(chunks, Chunk{
				ID:        generateChunkID(chunk.FilePath, currentStartLine, currentStartLine+lineCount-1, 0, uint32(len(chunks))),
				Content:   currentChunk.String(),
				FilePath:  chunk.FilePath,
				Language:  chunk.Language,
				StartLine: currentStartLine,
				EndLine:   currentStartLine + lineCount - 1,
				NodeType:  chunk.NodeType + "_part",
				Metadata:  chunk.Metadata,
			})

			currentChunk.Reset()
			currentStartLine += lineCount
			lineCount = 0
		} else {
			currentChunk.WriteString("\n")
		}
	}

	return chunks
}

// chunkGo extracts chunks from Go code
func (c *Chunker) chunkGo(node *sitter.Node, content []byte, filePath, language string) []Chunk {
	var chunks []Chunk

	// Extract package declaration
	if pkg := findFirstChildOfType(node, "package_clause"); pkg != nil {
		chunks = append(chunks, createChunk(pkg, content, filePath, language, "package_declaration"))
	}

	// Extract imports
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		if child.Type() == "import_declaration" {
			chunks = append(chunks, createChunk(child, content, filePath, language, "imports"))
		}
	}

	// Process top-level declarations
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration":
			// For functions, include the receiver if present
			receiver := findFirstChildOfType(child, "parameter_list")
			if receiver != nil && receiver.ChildCount() > 0 {
				receiverNode := receiver.Child(0)
				if receiverNode != nil && receiverNode.Type() == "parameter_declaration" {
					// This is a method, include the receiver in the chunk
					chunks = append(chunks, createChunk(child, content, filePath, language, "method_declaration"))
					continue
				}
			}
			chunks = append(chunks, createChunk(child, content, filePath, language, "function_declaration"))

		case "type_declaration":
			// For type declarations, check if it's a struct or interface
			typeSpec := findFirstChildOfType(child, "type_spec")
			if typeSpec != nil {
				typeNode := findFirstChildOfType(typeSpec, "struct_type", "interface_type")
				if typeNode != nil {
					chunks = append(chunks, createChunk(child, content, filePath, language, "type_"+typeNode.Type()))
				} else {
					chunks = append(chunks, createChunk(child, content, filePath, language, "type_declaration"))
				}
			}

		case "var_declaration", "const_declaration":
			// Group related vars/consts together
			chunks = append(chunks, createChunk(child, content, filePath, language, child.Type()))

		case "method_declaration":
			// Handle method declarations (though they should be inside type declarations)
			chunks = append(chunks, createChunk(child, content, filePath, language, "method_declaration"))
		}
	}

	return chunks
}

// chunkPython extracts chunks from Python code
func (c *Chunker) chunkPython(node *sitter.Node, content []byte, filePath, language string) []Chunk {
	var chunks []Chunk

	// Extract imports
	if imports := findFirstChildOfType(node, "import_statement", "import_from_statement"); imports != nil {
		chunks = append(chunks, createChunk(imports, content, filePath, language, "imports"))
	}

	// Extract top-level functions and classes
	n := int(node.ChildCount())
	for i := 0; i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_definition", "class_definition":
			chunks = append(chunks, createChunk(child, content, filePath, language, child.Type()))
		}
	}

	return chunks
}

// chunkJavaScript extracts chunks from JavaScript/TypeScript code
func (c *Chunker) chunkJavaScript(node *sitter.Node, content []byte, filePath, language string) []Chunk {
	var chunks []Chunk

	// Extract imports
	if imports := findFirstChildOfType(node, "import_statement", "import"); imports != nil {
		chunks = append(chunks, createChunk(imports, content, filePath, language, "imports"))
	}

	// Extract top-level functions, classes, and variable declarations
	n := int(node.ChildCount())
	for i := 0; i < n; i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		switch child.Type() {
		case "function_declaration", "class_declaration", "lexical_declaration":
			chunks = append(chunks, createChunk(child, content, filePath, language, child.Type()))
		}
	}

	return chunks
}

// chunkGeneric provides a generic chunking strategy for unsupported languages
func (c *Chunker) chunkGeneric(node *sitter.Node, content []byte, filePath, language string) []Chunk {
	// Just return the entire file as one chunk for unsupported languages
	return []Chunk{{
		ID:        generateChunkID(filePath, 1, bytesCountToLines(content), 0, 0),
		Content:   string(content),
		FilePath:  filePath,
		Language:  language,
		StartLine: 1,
		EndLine:   bytesCountToLines(content),
		NodeType:  "file",
	}}
}

// Helper function to create a chunk from a node
func createChunk(node *sitter.Node, content []byte, filePath, language, nodeType string) Chunk {
	startLine, endLine := GetNodePosition(node)
	return Chunk{
		ID:        generateChunkID(filePath, startLine, endLine, node.StartByte(), node.EndByte()),
		Content:   FormatNode(node, content),
		FilePath:  filePath,
		Language:  language,
		StartLine: startLine,
		EndLine:   endLine,
		NodeType:  nodeType,
		Metadata:  make(map[string]string),
	}
}

// Helper function to find the first child of any of the given types
func findFirstChildOfType(node *sitter.Node, types ...string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}

		for _, t := range types {
			if child.Type() == t {
				return child
			}
		}
	}
	return nil
}

// Helper function to count the number of lines in a byte slice
func bytesCountToLines(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	return bytes.Count(content, []byte("\n")) + 1
}

// Helper function to generate a unique ID for a chunk
func generateChunkID(filePath string, startLine, endLine int, startByte, endByte uint32) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d:%d:%d", filePath, startLine, endLine, startByte, endByte)))
	return hex.EncodeToString(hash[:])
}
