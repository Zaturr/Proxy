package build

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
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

// GenerateMockingbirdConfig genera la configuración de Mockingbird basada en múltiples BestMatch
func GenerateMockingbirdConfig(matches []BestMatch) *MockingbirdConfig {
	if len(matches) == 0 {
		return &MockingbirdConfig{}
	}

	host, port := extractHostAndPort(matches[0].URL)
	locations := createLocationsFromGroups(GroupSimilarRequests(matches))

	return &MockingbirdConfig{
		HTTP: HTTPConfig{
			Servers: []ServerConfig{{
				Listen:     port,
				Logger:     true,
				LoggerPath: "./logs/serverA",
				Name:       host,
				Version:    "0.0.1",
				Location:   locations,
			}},
		},
	}
}

// createLocationsFromGroups convierte grupos de requests en LocationConfig
func createLocationsFromGroups(groups map[string][]BestMatch) []LocationConfig {
	var locations []LocationConfig
	for _, group := range groups {
		if len(group) > 0 {
			first := group[0]
			locations = append(locations, LocationConfig{
				Path:       first.Endpoint,
				Method:     first.Method,
				Response:   first.Body,
				StatusCode: first.StatusCode,
				Headers:    first.Headers,
			})
		}
	}
	return locations
}

// extractHostAndPort extrae el host y puerto de una URL
func extractHostAndPort(url string) (string, int) {
	// Remover protocolo
	url = strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")

	// Extraer host y puerto
	parts := strings.Split(url, ":")
	host := parts[0]

	// Limpiar host si tiene path
	if strings.Contains(host, "/") {
		host = strings.Split(host, "/")[0]
	}

	// Extraer puerto si existe
	if len(parts) >= 2 {
		if portStr := strings.Split(parts[1], "/")[0]; portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				return host, port
			}
		}
	}

	// Puerto por defecto
	return host, 8080
}

// GroupSimilarRequests agrupa requests similares por endpoint y método
func GroupSimilarRequests(requests []BestMatch) map[string][]BestMatch {
	groups := make(map[string][]BestMatch)
	for _, request := range requests {
		key := request.Method + ":" + request.Endpoint
		groups[key] = append(groups[key], request)
	}
	return groups
}

// ToYAML convierte la configuración Mockingbird a formato YAML
func (config *MockingbirdConfig) ToYAML() (string, error) {
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(yamlData), nil
}
