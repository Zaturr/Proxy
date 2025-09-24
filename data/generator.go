package yaml

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	build "proxy/Build"
	"proxy/database"

	"gopkg.in/yaml.v3"
)

// GenerateYAMLFromDB genera YAML directamente desde la base de datos
func GenerateYAMLFromDB(db *sql.DB, endpoint, method string) (string, error) {
	// Obtener TODOS los requests de la base de datos para generar configuración completa
	fmt.Printf("Generating YAML configuration from all requests in database\n")
	dbMatches, err := database.GetAllRequestsForYAML(db)
	if err != nil {
		return "", fmt.Errorf("error getting all requests: %v", err)
	}

	fmt.Printf("Found %d total requests for YAML generation\n", len(dbMatches))
	if len(dbMatches) == 0 {
		return "", fmt.Errorf("no requests found in database")
	}

	// Convertir a database.BestMatch
	var buildMatches []database.BestMatch
	for _, dbMatch := range dbMatches {
		buildMatches = append(buildMatches, database.BestMatch{
			Endpoint:   dbMatch.Endpoint,
			Headers:    dbMatch.Headers,
			Body:       dbMatch.Body,
			StatusCode: dbMatch.StatusCode,
			Score:      dbMatch.Score,
			Method:     dbMatch.Method,
			URL:        dbMatch.URL,
			Port:       dbMatch.Port,
		})
	}

	// Generar configuración
	config := build.GenerateMockingbirdConfig(buildMatches)

	// Convertir a YAML
	yamlString, err := ToYAML(config)
	if err != nil {
		return "", fmt.Errorf("error generating YAML: %v", err)
	}

	// Guardar YAML en archivo
	fmt.Printf("DEBUG: About to save YAML to file\n")
	if err := saveYAMLToFile(yamlString, endpoint, method); err != nil {
		fmt.Printf("ERROR: Failed to save YAML to file: %v\n", err)
		return "", fmt.Errorf("error saving YAML to file: %v", err)
	}
	fmt.Printf("DEBUG: YAML saved successfully\n")

	return yamlString, nil
}

// saveYAMLToFile guarda el YAML en un archivo único consolidado
func saveYAMLToFile(yamlString, endpoint, method string) error {
	// Crear directorio yaml si no existe
	if err := os.MkdirAll("yaml", 0755); err != nil {
		return err
	}

	// Archivo único consolidado
	configFile := filepath.Join("yaml", "mockingbird_config.yaml")

	// Como ahora generamos configuración completa desde todos los requests,
	// simplemente reemplazamos el archivo completo
	fmt.Printf("DEBUG: Saving YAML to file: %s\n", configFile)
	fmt.Printf("DEBUG: YAML content length: %d\n", len(yamlString))
	if err := os.WriteFile(configFile, []byte(yamlString), 0644); err != nil {
		return err
	}

	fmt.Printf("YAML consolidado guardado en: %s\n", configFile)
	return nil
}

// consolidateConfigs combina configuraciones existentes con nuevas
func consolidateConfigs(existing, new *database.MockingbirdConfig) *database.MockingbirdConfig {
	// Si no hay configuración existente, usar la nueva
	if len(existing.HTTP.Servers) == 0 {
		return new
	}

	// Si no hay configuración nueva, usar la existente
	if len(new.HTTP.Servers) == 0 {
		return existing
	}

	// Obtener el primer servidor existente (asumimos un solo servidor)
	existingServer := &existing.HTTP.Servers[0]
	newServer := &new.HTTP.Servers[0]

	// Crear mapa para evitar duplicados basado en METHOD:PATH:RESPONSE
	existingLocations := make(map[string]database.LocationConfig)
	for _, loc := range existingServer.Location {
		key := loc.Method + ":" + loc.Path + ":" + loc.Response
		existingLocations[key] = loc
	}

	// Agregar nuevas ubicaciones solo si el response es diferente
	for _, newLoc := range newServer.Location {
		key := newLoc.Method + ":" + newLoc.Path + ":" + newLoc.Response
		// Solo agregar si no existe (response diferente)
		if _, exists := existingLocations[key]; !exists {
			existingLocations[key] = newLoc
			fmt.Printf("Nueva entrada agregada: %s %s (response diferente)\n", newLoc.Method, newLoc.Path)
		} else {
			fmt.Printf("Entrada duplicada ignorada: %s %s (mismo response)\n", newLoc.Method, newLoc.Path)
		}
	}

	// Convertir mapa de vuelta a slice
	var consolidatedLocations []database.LocationConfig
	for _, loc := range existingLocations {
		consolidatedLocations = append(consolidatedLocations, loc)
	}

	fmt.Printf("Consolidación completada: %d ubicaciones únicas\n", len(consolidatedLocations))

	// Crear configuración consolidada
	consolidatedConfig := &database.MockingbirdConfig{
		HTTP: database.HTTPConfig{
			Servers: []database.ServerConfig{{
				Listen:     existingServer.Listen,
				Logger:     existingServer.Logger,
				LoggerPath: existingServer.LoggerPath,
				Name:       existingServer.Name,
				Version:    existingServer.Version,
				Location:   consolidatedLocations,
			}},
		},
	}

	return consolidatedConfig
}

// ToYAML convierte la configuración Mockingbird a formato YAML
func ToYAML(config *database.MockingbirdConfig) (string, error) {
	yamlData, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(yamlData), nil
}
