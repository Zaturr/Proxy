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

// createLocationsFromGroups convierte grupos de requests en LocationConfig siguiendo las reglas estrictas
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

		// REGLA 1: Estructura Base (Siempre Presente)
		// Buscar la respuesta exitosa (status_code < 300) para la estructura base
		var baseResponse *database.BestMatch
		var errorResponse *database.BestMatch

		for _, match := range group {
			if match.StatusCode < 300 {
				baseResponse = &match
			} else if match.StatusCode >= 300 && errorResponse == nil {
				// REGLA 2: Solo tomar el primer código de error encontrado
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
			Path:        baseResponse.Endpoint,
			Method:      baseResponse.Method,
			Response:    baseResponse.Body,
			StatusCode:  baseResponse.StatusCode, // Código exitoso (ej. 200)
			ContentType: "application/json",
			Headers:     headers,
		}

		// REGLA 2: Inyección de Caos (Códigos de Error >= 300)
		// Solo agregar chaos_injection si hay un error >= 300
		if errorResponse != nil && errorResponse.StatusCode >= 300 {
			// Calcular probabilidad de caos basada en datos históricos
			probability, calculatedErrorCode, err := database.CalculateChaosProbability(db, baseResponse.URL)
			if err == nil && probability > 0 {
				location.ChaosInjection = &database.ChaosInjection{
					Probability: probability,
					StatusCode:  calculatedErrorCode, // Código de error para inyección de caos
				}
			} else {
				// Si no se puede calcular, usar el error encontrado directamente
				location.ChaosInjection = &database.ChaosInjection{
					Probability: 0.5, // Probabilidad por defecto
					StatusCode:  errorResponse.StatusCode,
				}
			}
		}
		// REGLA 3: Si no hay error >= 300, no se agrega chaos_injection

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
