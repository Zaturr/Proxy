package database

import (
	"time"
)

type ProxyRequest struct {
	ID        int64     `json:"id" db:"id"`
	Method    string    `json:"method" db:"method"`
	URL       string    `json:"url" db:"url"`
	Headers   string    `json:"headers" db:"headers"`
	Body      string    `json:"body" db:"body"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type ProxyResponse struct {
	ID         int64     `json:"id" db:"id"`
	RequestID  int64     `json:"request_id" db:"request_id"`
	StatusCode int       `json:"status_code" db:"status_code"`
	Headers    string    `json:"headers" db:"headers"`
	Body       string    `json:"body" db:"body"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

type ProxyTransaction struct {
	Request  ProxyRequest  `json:"request"`
	Response ProxyResponse `json:"response"`
}

type BestMatch struct {
	Endpoint   string            `json:"endpoint"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code"`
	Score      int               `json:"score"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Port       int               `json:"port"`
}

type SearchCriteria struct {
	URL      string            `json:"url"`
	Endpoint string            `json:"endpoint"`
	Body     string            `json:"body"`
	Headers  map[string]string `json:"headers"`
	Method   string            `json:"method"`
}

type MockingbirdConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	Listen     int              `yaml:"listen"`
	Logger     bool             `yaml:"logger"`
	LoggerPath string           `yaml:"logger_path"`
	Name       string           `yaml:"name"`
	Version    string           `yaml:"version"`
	Location   []LocationConfig `yaml:"location"`
}

type LocationConfig struct {
	Path       string            `yaml:"path"`
	Method     string            `yaml:"method"`
	Response   string            `yaml:"response"`
	StatusCode int               `yaml:"status_code"`
	Headers    map[string]string `yaml:"headers"`
	Schema     string            `yaml:"schema,omitempty"`
}
