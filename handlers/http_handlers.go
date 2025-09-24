package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	yaml "proxy/data"
	"proxy/database"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func getTargetProtocol(port string) string {

	httpsPorts := map[string]bool{
		// "8086": true,
		"8443": true,
		"443":  true,
	}

	if httpsPorts[port] {
		return "https"
	}
	return "http"
}

func MockingbirdProxy(db *sql.DB, c *gin.Context) {
	path := c.Param("path")

	var targetPort string
	switch {
	case strings.Contains(path, "/auth"):
		targetPort = "8086"
	case strings.Contains(path, "/hi"):
		targetPort = "8101" // Movido antes que /hello
	case strings.HasPrefix(path, "/jsonplaceholder"):
		targetPort = "8080"
	case strings.Contains(path, "/hello"):
		targetPort = "8080"
	case strings.Contains(path, "/echo"):
		targetPort = "8080"
	case strings.Contains(path, "/callback"):
		targetPort = "8080"
	case strings.HasPrefix(path, "/api"):
		targetPort = "8081" // API con HTTP
	default:
		targetPort = "8080" // Default
	}

	// Determinar protocolo automáticamente según el puerto
	protocol := getTargetProtocol(targetPort)
	targetURL := protocol + "://localhost:" + targetPort
	fmt.Printf("Path: %s, Target Port: %s, Protocol: %s, Target URL: %s\n", path, targetPort, protocol, targetURL)
	remote, _ := url.Parse(targetURL)

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(remote)
			r.Out.Host = r.In.Host // if desired
			body, err := io.ReadAll(r.In.Body)

			if err != nil {
				fmt.Printf("Error reading body: %s\n", err.Error())
			}
			// Guardar request en base de datos
			headersJSON, _ := json.Marshal(r.In.Header)
			portInt, _ := strconv.Atoi(targetPort)
			requestID, _ := database.InsertRequest(db, r.In.Method, r.In.URL.String(), string(headersJSON), string(body), portInt)
			c.Set("requestID", requestID)

			// Generar YAML automáticamente después de guardar request
			fmt.Printf("Calling GenerateYAMLFromDB for %s %s\n", r.In.Method, r.In.URL.Path)
			go func() {
				_, err := yaml.GenerateYAMLFromDB(db, r.In.URL.Path, r.In.Method)
				if err != nil {
					fmt.Printf("Error generating YAML automatically: %v\n", err)
				}
			}()

			fmt.Printf("Request body: %s\n", body)
			r.Out.Body = io.NopCloser(bytes.NewBuffer(body))
		},
		ModifyResponse: func(r *http.Response) error {
			body, err := io.ReadAll(r.Body)

			if err != nil {
				return err
			}

			// Guardar response en base de datos
			if requestID, exists := c.Get("requestID"); exists {
				headersJSON, _ := json.Marshal(r.Header)
				database.InsertResponse(db, requestID.(int64), r.StatusCode, string(headersJSON), string(body))
			}

			fmt.Printf("Request body: %s\n", body)
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			return nil
		},
	}

	//si no hay servidor de destino devolverá 404
	proxy.ServeHTTP(c.Writer, c.Request)
}

func SearchHandler(db *sql.DB, c *gin.Context) {
	var criteria database.SearchCriteria

	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	bestMatch, err := database.HierarchicalSearch(db, criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	if bestMatch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No matching records found"})
		return
	}

	c.JSON(http.StatusOK, bestMatch)
}

// simulateResponse simula una respuesta cuando no hay servidor de destino
func SimulateResponse(db *sql.DB, c *gin.Context) {
	headersJSON, _ := json.Marshal(c.Request.Header)
	body, _ := io.ReadAll(c.Request.Body)

	// Determinar puerto basado en el path (misma lógica que MockingbirdProxy)
	path := c.Request.URL.Path
	var targetPort string
	switch {
	case strings.Contains(path, "/auth"):
		targetPort = "8086"
	case strings.Contains(path, "/hi"):
		targetPort = "8101"
	case strings.HasPrefix(path, "/jsonplaceholder"):
		targetPort = "8080"
	case strings.Contains(path, "/hello"):
		targetPort = "8080"
	case strings.Contains(path, "/echo"):
		targetPort = "8080"
	case strings.Contains(path, "/callback"):
		targetPort = "8080"
	case strings.HasPrefix(path, "/api"):
		targetPort = "8081"
	default:
		targetPort = "8080"
	}

	portInt, _ := strconv.Atoi(targetPort)
	requestID, _ := database.InsertRequest(db, c.Request.Method, c.Request.URL.String(), string(headersJSON), string(body), portInt)

	responseBody := `{"message": "Simulated response", "status": "ok"}`
	responseHeaders := map[string]string{
		"Content-Type": "application/json",
	}

	responseHeadersJSON, _ := json.Marshal(responseHeaders)
	database.InsertResponse(db, requestID, 200, string(responseHeadersJSON), responseBody)

	go func() {
		_, err := yaml.GenerateYAMLFromDB(db, c.Request.URL.Path, c.Request.Method)
		if err != nil {
			fmt.Printf("Error generating YAML automatically: %v\n", err)
		}
	}()

	c.Header("Content-Type", "application/json")
	c.JSON(200, gin.H{
		"message": "Simulated response",
		"status":  "ok",
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	})
}

// StatsHandler maneja las estadísticas de requests
func StatsHandler(db *sql.DB, c *gin.Context) {
	stats, err := yaml.GetRequestStats(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// CountHandler maneja los conteos específicos
func CountHandler(db *sql.DB, c *gin.Context) {
	countType := c.Query("type")

	switch countType {
	case "approved":
		counts, err := yaml.CountApprovedRequests(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count approved requests"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"approved_requests": counts})

	case "rejected":
		counts, err := yaml.CountRejectedRequests(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count rejected requests"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"rejected_requests": counts})

	case "yaml_paths":
		paths, err := yaml.GetUniquePathsForYAML(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get YAML paths"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"yaml_paths": paths})

	case "error_paths":
		paths, err := yaml.GetErrorPathsForChaosInjection(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get error paths"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"error_paths": paths})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid count type. Use: approved, rejected, yaml_paths, error_paths"})
	}
}
