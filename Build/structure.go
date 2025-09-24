package build

import (
	"fmt"
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

	// Agrupar por puerto para crear múltiples servidores
	portGroups := GroupByPort(matches)
	var servers []database.ServerConfig

	fmt.Printf("DEBUG: Found %d unique ports in matches\n", len(portGroups))
	for port, portMatches := range portGroups {
		fmt.Printf("DEBUG: Port %d has %d matches\n", port, len(portMatches))
		for i, match := range portMatches {
			fmt.Printf("DEBUG: Port %d, Match %d: %s %s (Port: %d)\n", port, i, match.Method, match.Endpoint, match.Port)
		}

		locations := createLocationsFromGroups(GroupSimilarRequests(portMatches))

		servers = append(servers, database.ServerConfig{
			Listen:     port,
			Logger:     true,
			LoggerPath: fmt.Sprintf("./logs/server_%d", port),
			Name:       fmt.Sprintf("server_%d", port),
			Version:    "0.0.1",
			Location:   locations,
		})
	}

	return &database.MockingbirdConfig{
		HTTP: database.HTTPConfig{
			Servers: servers,
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

// GroupByPort agrupa los requests por puerto de destino
func GroupByPort(requests []database.BestMatch) map[int][]database.BestMatch {
	groups := make(map[int][]database.BestMatch)
	for _, request := range requests {
		groups[request.Port] = append(groups[request.Port], request)
	}
	return groups
}

func GroupSimilarRequests(requests []database.BestMatch) map[string][]database.BestMatch {
	groups := make(map[string][]database.BestMatch)
	for _, request := range requests {
		key := request.Method + ":" + request.Endpoint
		groups[key] = append(groups[key], request)
	}
	return groups
}
