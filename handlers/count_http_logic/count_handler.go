package count_http_logic

import (
	"database/sql"
	"net/http"
	yaml "proxy/data"

	"github.com/gin-gonic/gin"
)

// CountHandler maneja los conteos específicos
func CountHandler(db *sql.DB, c *gin.Context) {
	countType := c.Query("type")

	switch countType {
	case "approved":
		handleApprovedCount(db, c)
	case "rejected":
		handleRejectedCount(db, c)
	case "yaml_paths":
		handleYAMLPaths(db, c)
	case "error_paths":
		handleErrorPaths(db, c)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid count type. Use: approved, rejected, yaml_paths, error_paths"})
	}
}

// handleApprovedCount maneja el conteo de requests aprobados
func handleApprovedCount(db *sql.DB, c *gin.Context) {
	counts, err := yaml.CountApprovedRequests(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count approved requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"approved_requests": counts})
}

// handleRejectedCount maneja el conteo de requests rechazados
func handleRejectedCount(db *sql.DB, c *gin.Context) {
	counts, err := yaml.CountRejectedRequests(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count rejected requests"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rejected_requests": counts})
}

// handleYAMLPaths maneja la obtención de paths únicos para YAML
func handleYAMLPaths(db *sql.DB, c *gin.Context) {
	paths, err := yaml.GetUniquePathsForYAML(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get YAML paths"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"yaml_paths": paths})
}

// handleErrorPaths maneja la obtención de paths de error para chaos injection
func handleErrorPaths(db *sql.DB, c *gin.Context) {
	paths, err := yaml.GetErrorPathsForChaosInjection(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get error paths"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error_paths": paths})
}
