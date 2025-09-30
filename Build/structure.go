package build

import (
	"database/sql"
	"fmt"
	"strings"

	"proxy/database"
	infra "proxy/database/infraestructure"
)

// ProxyPort es el puerto donde corre el proxy (debe coincidir con main.go)
const ProxyPort = 3000

// GenerateMockingbirdConfig genera la configuración de Mockingbird basada en múltiples BestMatch
func GenerateMockingbirdConfig(matches []database.BestMatch, db *sql.DB) *database.MockingbirdConfig {
	if len(matches) == 0 {
		return &database.MockingbirdConfig{}
	}

	portGroups := GroupByPort(matches)
	var servers []database.ServerConfig

	for port, portMatches := range portGroups {
		locations := createLocationsFromGroups(GroupSimilarRequests(portMatches), db)

		// Calcular ChaosInjection a nivel de servidor
		var chaosInjection *database.ChaosInjection
		if len(portMatches) > 0 {
			// Usar el primer match para calcular la probabilidad de caos
			firstMatch := portMatches[0]
			probability, calculatedErrorCode, err := infra.CalculateChaosProbability(db, firstMatch.URL)
			if err == nil && probability > 0 {
				chaosInjection = &database.ChaosInjection{
					Probability: probability,
					StatusCode:  calculatedErrorCode,
				}
			}
		}

		servers = append(servers, database.ServerConfig{
			Listen:         port,
			Logger:         true,
			LoggerPath:     fmt.Sprintf("./logs/server_%d", port),
			Name:           fmt.Sprintf("server_%d", port),
			Version:        "0.0.1",
			Location:       locations,
			ChaosInjection: chaosInjection,
		})
	}

	return &database.MockingbirdConfig{
		HTTP: database.HTTPConfig{
			Servers: servers,
		},
	}
}

func createLocationsFromGroups(groups map[string][]database.BestMatch, db *sql.DB) []database.LocationConfig {
	var locations []database.LocationConfig

	// Agrupar por path+method para manejar múltiples escenarios
	pathMethodGroups := make(map[string][]database.BestMatch)
	for _, group := range groups {
		if len(group) > 0 {
			first := group[0]
			key := first.Method + ":" + first.Endpoint
			pathMethodGroups[key] = append(pathMethodGroups[key], group...)
		}
	}

	for _, group := range pathMethodGroups {
		if len(group) == 0 {
			continue
		}

		var baseResponse *database.BestMatch
		var errorResponse *database.BestMatch

		for _, match := range group {
			if match.StatusCode < 300 {
				baseResponse = &match
			} else if match.StatusCode >= 300 && errorResponse == nil {

				errorResponse = &match
			}
		}

		// Si no hay respuesta exitosa, usar la primera disponible
		if baseResponse == nil {
			baseResponse = &group[0]
		}

		// Crear headers sin Content-Type (se maneja por separado)
		headers := make(map[string]string)
		if baseResponse.Headers != nil {
			for k, v := range baseResponse.Headers {
				// No incluir Content-Type en headers, se maneja por separado
				if k != "Content-Type" {
					headers[k] = v
				}
			}
		}

		// Estructura base siempre presente
		location := database.LocationConfig{
			Path:       baseResponse.Endpoint,
			Method:     baseResponse.Method,
			Response:   baseResponse.Body,
			StatusCode: baseResponse.StatusCode, // Código exitoso (ej. 200)
			Headers: &database.Headers{
				ContentType: "application/json",
			},
		}

		locations = append(locations, location)
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
