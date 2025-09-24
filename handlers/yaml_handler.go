package handlers

import (
	"database/sql"
	"net/http"
	yaml "proxy/data"
	"proxy/database"

	"github.com/gin-gonic/gin"
)

func SearchConfigHandler(db *sql.DB, c *gin.Context) {
	var criteria database.SearchCriteria

	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	yamlString, err := yaml.GenerateYAMLFromDB(db, criteria.Endpoint, criteria.Method)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"yaml": yamlString})
}
