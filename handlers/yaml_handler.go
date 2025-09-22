package handlers

import (
	"database/sql"
	"net/http"
	"proxy/database"
	"proxy/yaml"

	"github.com/gin-gonic/gin"
)

// SearchConfigHandler maneja la generación de configuración YAML
func SearchConfigHandler(db *sql.DB, c *gin.Context) {
	var criteria database.SearchCriteria

	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Usar la función centralizada para generar YAML
	yamlString, err := yaml.GenerateYAMLFromDB(db, criteria.Endpoint, criteria.Method)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Retornar el YAML como string
	c.JSON(http.StatusOK, gin.H{"yaml": yamlString})
}
