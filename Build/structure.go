package build

import (
	"strconv"
	"strings"
)

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

// GenerateMockingbirdConfig genera la configuración de Mockingbird basada en BestMatch
func GenerateMockingbirdConfig(bestMatch *BestMatch) *MockingbirdConfig {
	// Extraer host y puerto de la URL
	host, port := extractHostAndPort(bestMatch.URL)

	// Crear la configuración
	config := &MockingbirdConfig{
		HTTP: HTTPConfig{
			Servers: []ServerConfig{
				{
					Listen:     port,
					Logger:     true,
					LoggerPath: "./logs/serverA",
					Name:       host,
					Version:    "0.0.1",
					Location: []LocationConfig{
						{
							Path:       bestMatch.Endpoint,
							Method:     bestMatch.Method,
							Response:   bestMatch.Body,
							StatusCode: bestMatch.StatusCode,
							Headers:    bestMatch.Headers,
						},
					},
				},
			},
		},
	}

	return config
}

// extractHostAndPort extrae el host y puerto de una URL
func extractHostAndPort(url string) (string, int) {
	// Remover protocolo si existe
	if strings.HasPrefix(url, "http://") {
		url = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		url = strings.TrimPrefix(url, "https://")
	}

	// Dividir por : para separar host y puerto
	parts := strings.Split(url, ":")
	if len(parts) >= 2 {
		// Extraer puerto
		portStr := strings.Split(parts[1], "/")[0] // Remover path si existe
		if port, err := strconv.Atoi(portStr); err == nil {
			return parts[0], port
		}
	}

	// Si no se encuentra puerto, usar valores por defecto
	host := parts[0]
	if strings.Contains(host, "/") {
		host = strings.Split(host, "/")[0]
	}

	// Puerto por defecto basado en si es localhost
	if host == "localhost" || host == "127.0.0.1" {
		return host, 8080
	}

	return host, 8080 // Puerto por defecto para todos los casos
}
