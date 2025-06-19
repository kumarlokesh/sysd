package indexer

import (
	"path/filepath"
	"strings"
)

// FileType represents a programming language and its associated file extensions
type FileType struct {
	Name       string
	Extensions []string
}

// DefaultFileTypes contains the supported programming languages and their file extensions
var DefaultFileTypes = []FileType{
	{"go", []string{".go"}},
	{"python", []string{".py"}},
	{"javascript", []string{".js"}},
	{"typescript", []string{".ts"}},
	{"typescript", []string{".tsx"}},
	{"javascript", []string{".jsx"}},
	{"java", []string{".java"}},
	{"c", []string{".c", ".h"}},
	{"cpp", []string{".cpp", ".hpp", ".cc", ".cxx", ".hxx"}},
	{"rust", []string{".rs"}},
	{"ruby", []string{".rb"}},
}

// DefaultLanguageDetector is the default implementation of LanguageDetector
type DefaultLanguageDetector struct {
	extensionMap map[string]string
}

// NewDefaultLanguageDetector creates a new DefaultLanguageDetector
func NewDefaultLanguageDetector() *DefaultLanguageDetector {
	extMap := make(map[string]string)
	for _, ft := range DefaultFileTypes {
		for _, ext := range ft.Extensions {
			extMap[ext] = ft.Name
		}
	}
	return &DefaultLanguageDetector{extensionMap: extMap}
}

// Detect detects the programming language of a file based on its extension
func (d *DefaultLanguageDetector) Detect(path string, _ []byte) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := d.extensionMap[ext]; ok {
		return lang, nil
	}
	return "", nil
}

// GetSupportedLanguages returns a list of supported programming languages
func (d *DefaultLanguageDetector) GetSupportedLanguages() []string {
	languages := make(map[string]bool)
	for _, lang := range d.extensionMap {
		languages[lang] = true
	}

	result := make([]string, 0, len(languages))
	for lang := range languages {
		result = append(result, lang)
	}
	return result
}
