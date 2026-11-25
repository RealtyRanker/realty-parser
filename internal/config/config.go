package config

import (
	"os"

	"github.com/asmisnik/realty-parser/internal/model"
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

func (cfg SearchConfig) MatchesFlat(f *model.FlatInfo) bool {
	if f.Price < cfg.MinPrice || (cfg.MaxPrice > 0 && f.Price > cfg.MaxPrice) ||
		f.TotalArea < float64(cfg.MinArea) || (cfg.MaxArea > 0 && f.TotalArea > float64(cfg.MaxArea)) {
		return false
	}
	if len(cfg.Rooms) == 0 {
		return true
	}
	for _, room := range cfg.Rooms {
		if (room <= 5 && f.RoomNumber == room-1) || (room == 6 && f.RoomNumber >= 5) {
			return true
		}
	}
	return false
}

type WorkerConfig struct {
	PeriodSeconds        int     `yaml:"period_seconds"`
	RequestTimeoutMs     int     `yaml:"request_timeout_ms"`
	SleepBeforeRequestMs int     `yaml:"sleep_before_request_ms"`
	RetrySleepMs         int     `yaml:"retry_sleep_ms"`
	RetrySleepMultiplier float64 `yaml:"retry_sleep_multiplier"`
	MaxRetries           int     `yaml:"max_retries"`
	CookieHeader         string  `yaml:"cookie_header"`
	// DetailSource: "flat" (default) — fetch every flat page; "search" — take listings
	// straight from the search results page (no per-flat requests, fewer fields).
	DetailSource string `yaml:"detail_source"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

type YandexConfig struct {
	Enabled bool        `yaml:"enabled"`
	Regions map[int]int `yaml:"regions"` // RealtyRanker region ID -> Yandex Realty rgid
}

type DomclickRegion struct {
	Address string `yaml:"address"`
	AID     int    `yaml:"aid"`
}

type DomclickConfig struct {
	Enabled bool                   `yaml:"enabled"`
	Regions map[int]DomclickRegion `yaml:"regions"`
}

type LoggingConfig struct {
	FilePath string `yaml:"file_path"`
	Level    string `yaml:"level"`
}

type MetricsConfig struct {
	Port int `yaml:"port"`
}

type Config struct {
	Search   SearchConfig   `yaml:"search"`
	Yandex   YandexConfig   `yaml:"yandex"`
	Domclick DomclickConfig `yaml:"domclick"`
	Worker   WorkerConfig   `yaml:"worker"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LoggingConfig  `yaml:"logging"`
	Metrics  MetricsConfig  `yaml:"metrics"`
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
