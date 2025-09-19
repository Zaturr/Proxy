package yaml

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	build "proxy/Build"
	"proxy/database"
	"strings"
	"time"
)

// GenerateYAMLFromDB genera YAML directamente desde la base de datos
func GenerateYAMLFromDB(db *sql.DB, endpoint, method string) (string, error) {
	// Crear criterios de búsqueda
	criteria := database.SearchCriteria{
		Endpoint: endpoint,
		Method:   method,
	}

	// Obtener requests similares
	fmt.Printf("Searching for endpoint: %s, method: %s\n", endpoint, method)
	dbMatches, err := database.GetSimilarRequests(db, criteria)
	if err != nil {
		return "", fmt.Errorf("error getting similar requests: %v", err)
	}

	fmt.Printf("Found %d similar requests\n", len(dbMatches))
	if len(dbMatches) == 0 {
		return "", fmt.Errorf("no matching records found")
	}

	// Convertir a build.BestMatch
	var buildMatches []build.BestMatch
	for _, dbMatch := range dbMatches {
		buildMatches = append(buildMatches, build.BestMatch{
			Endpoint:   dbMatch.Endpoint,
			Headers:    dbMatch.Headers,
			Body:       dbMatch.Body,
			StatusCode: dbMatch.StatusCode,
			Score:      dbMatch.Score,
			Method:     dbMatch.Method,
			URL:        dbMatch.URL,
		})
	}

	// Generar configuración
	config := build.GenerateMockingbirdConfig(buildMatches)

	// Convertir a YAML
	yamlString, err := config.ToYAML()
	if err != nil {
		return "", fmt.Errorf("error generating YAML: %v", err)
	}

	// Guardar YAML en archivo
	if err := saveYAMLToFile(yamlString, endpoint, method); err != nil {
		return "", fmt.Errorf("error saving YAML to file: %v", err)
	}

	return yamlString, nil
}

// saveYAMLToFile guarda el YAML en un archivo
func saveYAMLToFile(yamlString, endpoint, method string) error {
	// Crear directorio yaml si no existe
	if err := os.MkdirAll("yaml", 0755); err != nil {
		return err
	}

	// Crear nombre de archivo con timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("mockingbird_%s_%s_%s.yaml",
		strings.ReplaceAll(endpoint, "/", "_"),
		method,
		timestamp)

	filepath := filepath.Join("yaml", filename)

	// Escribir archivo
	if err := os.WriteFile(filepath, []byte(yamlString), 0644); err != nil {
		return err
	}

	fmt.Printf("YAML guardado en: %s\n", filepath)
	return nil
}
