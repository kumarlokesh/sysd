package config

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	ChromaDB  ChromaDBConfig  `mapstructure:"chromadb"`
	LLM       LLMConfig       `mapstructure:"llm"`
	Embedding EmbeddingConfig `mapstructure:"embeddings"`
	Context   ContextConfig   `mapstructure:"context"`
	Cache     CacheConfig     `mapstructure:"cache"`
}

// ServerConfig holds server related configuration
type ServerConfig struct {
	Port  int    `mapstructure:"port"`
	Host  string `mapstructure:"host"`
	Debug bool   `mapstructure:"debug"`
}

// ChromaDBConfig holds ChromaDB related configuration
type ChromaDBConfig struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"`
}

// LLMConfig holds LLM related configuration
type LLMConfig struct {
	Model       string        `mapstructure:"model"`
	Temperature float64       `mapstructure:"temperature"`
	MaxTokens   int           `mapstructure:"max_tokens"`
	Timeout     time.Duration `mapstructure:"timeout"`
}

// EmbeddingConfig holds embedding related configuration
type EmbeddingConfig struct {
	Model     string `mapstructure:"model"`
	BatchSize int    `mapstructure:"batch_size"`
	VectorDim int    `mapstructure:"vector_dimension"`
}

// ContextConfig holds context related configuration
type ContextConfig struct {
	MaxFiles     int `mapstructure:"max_files"`
	MaxFileSize  int `mapstructure:"max_file_size"`
	ChunkSize    int `mapstructure:"chunk_size"`
	ChunkOverlap int `mapstructure:"chunk_overlap"`
}

// CacheConfig holds cache related configuration
type CacheConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	TTL     time.Duration `mapstructure:"ttl"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	setDefaults(v)

	if configPath != "" {
		v.SetConfigFile(configPath)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	v.AutomaticEnv()

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.debug", true)
	v.SetDefault("chromadb.url", "http://localhost:8000")
	v.SetDefault("chromadb.api_key", "")

	// LLM defaults
	v.SetDefault("llm.model", "codellama:7b")
	v.SetDefault("llm.temperature", 0.2)
	v.SetDefault("llm.max_tokens", 2048)
	v.SetDefault("llm.timeout", "5m")

	// Embedding defaults
	v.SetDefault("embeddings.model", "sentence-transformers/all-mpnet-base-v2")
	v.SetDefault("embeddings.batch_size", 32)
	v.SetDefault("embeddings.vector_dimension", 768)

	// Context defaults
	v.SetDefault("context.max_files", 50)
	v.SetDefault("context.max_file_size", 1000000) // 1MB
	v.SetDefault("context.chunk_size", 1000)
	v.SetDefault("context.chunk_overlap", 200)

	// Cache defaults
	v.SetDefault("cache.enabled", true)
	v.SetDefault("cache.ttl", "24h")
}

// GetURL returns the ChromaDB server URL
func (c *ChromaDBConfig) GetURL() string {
	return c.URL
}

// GetAPIKey returns the ChromaDB API key if set
func (c *ChromaDBConfig) GetAPIKey() string {
	return c.APIKey
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.ChromaDB.URL == "" {
		return fmt.Errorf("chromadb url is required")
	}
	if c.ChromaDB.APIKey == "" {
		return fmt.Errorf("chromadb api key is required")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.LLM.Model == "" {
		return fmt.Errorf("LLM model cannot be empty")
	}

	return nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	// Look for config in the following locations:
	// 1. Current directory
	// 2. ./configs/
	// 3. /etc/ai-code-assistant/

	configName := "config"
	configType := "yaml"
	configPaths := []string{
		".",
		"./configs",
		"/etc/ai-code-assistant",
	}

	for _, path := range configPaths {
		configPath := filepath.Join(path, configName+"."+configType)
		if _, err := filepath.Abs(configPath); err == nil {
			return configPath, nil
		}
	}

	return "", fmt.Errorf("config file not found in any of the default locations")
}
