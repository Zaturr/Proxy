package build

import (
	"strconv"
	"strings"

	"proxy/database"
)

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

// extractHostAndPort extrae el host y puerto de una URL
func extractHostAndPort(url string) (string, int) {
	url = strings.TrimPrefix(strings.TrimPrefix(url, "http://"), "https://")

	parts := strings.Split(url, ":")
	host := parts[0]

	if strings.Contains(host, "/") {
		host = strings.Split(host, "/")[0]
	}

	if len(parts) >= 2 {
		if portStr := strings.Split(parts[1], "/")[0]; portStr != "" {
			if port, err := strconv.Atoi(portStr); err == nil {
				return host, port
			}
		}
	}

	return host, 8080
}

func GroupSimilarRequests(requests []database.BestMatch) map[string][]database.BestMatch {
	groups := make(map[string][]database.BestMatch)
	for _, request := range requests {
		key := request.Method + ":" + request.Endpoint
		groups[key] = append(groups[key], request)
	}
	return groups
}
