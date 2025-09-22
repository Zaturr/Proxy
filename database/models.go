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

// BestMatch representa el mejor candidato encontrado en la búsqueda jerárquica
type BestMatch struct {
	Endpoint   string            `json:"endpoint"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code"`
	Score      int               `json:"score"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
}

// SearchCriteria representa los criterios de búsqueda
type SearchCriteria struct {
	URL      string            `json:"url"`
	Endpoint string            `json:"endpoint"`
	Body     string            `json:"body"`
	Headers  map[string]string `json:"headers"`
	Method   string            `json:"method"`
}

// MockingbirdConfig representa la estructura completa de configuración de Mockingbird
type MockingbirdConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

// HTTPConfig representa la configuración HTTP
type HTTPConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

// ServerConfig representa la configuración de un servidor
type ServerConfig struct {
	Listen     int              `yaml:"listen"`
	Logger     bool             `yaml:"logger"`
	LoggerPath string           `yaml:"logger_path"`
	Name       string           `yaml:"name"`
	Version    string           `yaml:"version"`
	Location   []LocationConfig `yaml:"location"`
}

// LocationConfig representa la configuración de una ubicación/endpoint
type LocationConfig struct {
	Path       string            `yaml:"path"`
	Method     string            `yaml:"method"`
	Response   string            `yaml:"response"`
	StatusCode int               `yaml:"status_code"`
	Headers    map[string]string `yaml:"headers"`
	Schema     string            `yaml:"schema,omitempty"`
}
