package build

import (
	"database/sql"
	"fmt"
	"strings"

	"proxy/database"
)

// ProxyPort es el puerto donde corre el proxy (debe coincidir con main.go)
const ProxyPort = 3000

// GenerateMockingbirdConfig genera la configuración de Mockingbird basada en múltiples BestMatch
func GenerateMockingbirdConfig(matches []database.BestMatch, db *sql.DB) *database.MockingbirdConfig {
	if len(matches) == 0 {
		return &database.MockingbirdConfig{}
	}

	// Agrupar por puerto para crear múltiples servidores
	portGroups := GroupByPort(matches)
	var servers []database.ServerConfig

	for port, portMatches := range portGroups {
		locations := createLocationsFromGroups(GroupSimilarRequests(portMatches), db)

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
func createLocationsFromGroups(groups map[string][]database.BestMatch, db *sql.DB) []database.LocationConfig {
	var locations []database.LocationConfig
	for _, group := range groups {
		if len(group) > 0 {
			first := group[0]

			// Crear headers sin Content-Type (se maneja por separado)
			headers := make(map[string]string)
			if first.Headers != nil {
				for k, v := range first.Headers {
					// No incluir Content-Type en headers, se maneja por separado
					if k != "Content-Type" {
						headers[k] = v
					}
				}
			}

			location := database.LocationConfig{
				Path:        first.Endpoint,
				Method:      first.Method,
				Response:    first.Body,
				StatusCode:  first.StatusCode,
				ContentType: "application/json",
				Headers:     headers,
			}

			// El chaos injection es un extra que se agrega al código aprobado
			// Se calcula basándose en la probabilidad de errores históricos para esta URL
			probability, errorStatusCode, err := database.CalculateChaosProbability(db, first.URL)
			if err == nil && probability > 0 {
				location.ChaosInjection = &database.ChaosInjection{
					Probability: probability,
					StatusCode:  errorStatusCode,
				}
			}

			locations = append(locations, location)
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
		// Incluir status code en la clave para separar requests exitosos de los que tienen errores
		key := request.Method + ":" + request.Endpoint + ":" + fmt.Sprintf("%d", request.StatusCode)
		groups[key] = append(groups[key], request)
	}
	return groups
}
