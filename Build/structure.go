package build

import (
	"strings"

	"proxy/database"
)

// ProxyPort es el puerto donde corre el proxy (debe coincidir con main.go)
const ProxyPort = 3000

// GenerateMockingbirdConfig genera la configuración de Mockingbird basada en múltiples BestMatch
func GenerateMockingbirdConfig(matches []database.BestMatch) *database.MockingbirdConfig {
	if len(matches) == 0 {
		return &database.MockingbirdConfig{}
	}

	host, port := extractHostAndPort(matches[0].URL)
	locations := createLocationsFromGroups(GroupSimilarRequests(matches))

	return &database.MockingbirdConfig{
		HTTP: database.HTTPConfig{
			Servers: []database.ServerConfig{{
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
func createLocationsFromGroups(groups map[string][]database.BestMatch) []database.LocationConfig {
	var locations []database.LocationConfig
	for _, group := range groups {
		if len(group) > 0 {
			first := group[0]
			locations = append(locations, database.LocationConfig{
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

func extractHostAndPort(url string) (string, int) {
	url = strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")

	parts := strings.Split(url, ":")
	host := parts[0]

	// Limpiar host si tiene path
	if strings.Contains(host, "/") {
		host = strings.Split(host, "/")[0]
	}

	return host, ProxyPort
}

func GroupSimilarRequests(requests []database.BestMatch) map[string][]database.BestMatch {
	groups := make(map[string][]database.BestMatch)
	for _, request := range requests {
		key := request.Method + ":" + request.Endpoint
		groups[key] = append(groups[key], request)
	}
	return groups
}
