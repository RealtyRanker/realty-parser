package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type SearchConfig struct {
	MinPrice     int    `yaml:"min_price"`
	MaxPrice     int    `yaml:"max_price"`
	MinArea      int    `yaml:"min_area"`
	MaxArea      int    `yaml:"max_area"`
	Sort         string `yaml:"sort"`
	Rooms        []int  `yaml:"rooms"`
	PagesLimit   int    `yaml:"pages_limit"`
	Limit        int    `yaml:"limit"`
	OverallLimit int    `yaml:"overall_limit"`
}

type WorkerConfig struct {
	PeriodSeconds        int     `yaml:"period_seconds"`
	RequestTimeoutMs     int     `yaml:"request_timeout_ms"`
	SleepBeforeRequestMs int     `yaml:"sleep_before_request_ms"`
	RetrySleepMs         int     `yaml:"retry_sleep_ms"`
	RetrySleepMultiplier float64 `yaml:"retry_sleep_multiplier"`
	MaxRetries           int     `yaml:"max_retries"`
	CookieHeader         string  `yaml:"cookie_header"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type LoggingConfig struct {
	FilePath string `yaml:"file_path"`
	Level    string `yaml:"level"`
}

type MetricsConfig struct {
	Port int `yaml:"port"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic"`
}

type Config struct {
	Search   SearchConfig   `yaml:"search"`
	Worker   WorkerConfig   `yaml:"worker"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Kafka    KafkaConfig    `yaml:"kafka"`
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
	return &cfg, nil
}
