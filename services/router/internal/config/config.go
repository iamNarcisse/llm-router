package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	Qdrant    QdrantConfig    `yaml:"qdrant"`
	Routing   RoutingConfig   `yaml:"routing"`
}

type ServerConfig struct {
	GRPCPort int `yaml:"grpc_port"`
	HTTPPort int `yaml:"http_port"`
}

type EmbeddingConfig struct {
	Address string        `yaml:"address"`
	Timeout time.Duration `yaml:"timeout"`
}

type QdrantConfig struct {
	Host       string `yaml:"host"`
	Port       int    `yaml:"port"`
	Collection string `yaml:"collection"`
}

type RoutingConfig struct {
	DefaultModel   string  `yaml:"default_model"`
	ScoreThreshold float32 `yaml:"score_threshold"`
	TopK           int     `yaml:"top_k"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Apply defaults
	if cfg.Routing.TopK == 0 {
		cfg.Routing.TopK = 3
	}
	if cfg.Routing.ScoreThreshold == 0 {
		cfg.Routing.ScoreThreshold = 0.5
	}
	if cfg.Qdrant.Port == 0 {
		cfg.Qdrant.Port = 6334
	}

	return &cfg, nil
}
