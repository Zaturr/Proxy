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
	"proxy/database"
	"proxy/yaml"
	"strings"

	"github.com/gin-gonic/gin"
)

func getTargetProtocol(port string) string {

	httpsPorts := map[string]bool{
		"8086": true,
		"8443": true,
		"443":  true,
	}

	if httpsPorts[port] {
		return "https"
	}
	return "http"
}

func ProxyHandler(c *gin.Context) {
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL parameter required"})
		return
	}

	remote, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL"})
		return
	}

	httputil.NewSingleHostReverseProxy(remote).ServeHTTP(c.Writer, c.Request)
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
			requestID, _ := database.InsertRequest(db, r.In.Method, r.In.URL.String(), string(headersJSON), string(body))
			c.Set("requestID", requestID)

			// Generar YAML automáticamente después de guardar request
			fmt.Printf("Calling generateYAMLAutomatically for %s %s\n", r.In.Method, r.In.URL.Path)
			go generateYAMLAutomatically(db, r.In.URL.Path, r.In.Method)

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

// generateYAMLAutomatically genera YAML automáticamente después de cada request
func generateYAMLAutomatically(db *sql.DB, path, method string) {
	fmt.Printf("Starting automatic YAML generation for %s %s\n", method, path)

	// Generar YAML para este endpoint
	_, err := yaml.GenerateYAMLFromDB(db, path, method)
	if err != nil {
		fmt.Printf("Error generating YAML automatically: %v\n", err)
		return
	}

	fmt.Printf("YAML generated automatically for %s %s\n", method, path)
}

// simulateResponse simula una respuesta cuando no hay servidor de destino
func SimulateResponse(db *sql.DB, c *gin.Context) {
	headersJSON, _ := json.Marshal(c.Request.Header)
	body, _ := io.ReadAll(c.Request.Body)
	requestID, _ := database.InsertRequest(db, c.Request.Method, c.Request.URL.String(), string(headersJSON), string(body))

	responseBody := `{"message": "Simulated response", "status": "ok"}`
	responseHeaders := map[string]string{
		"Content-Type": "application/json",
	}

	responseHeadersJSON, _ := json.Marshal(responseHeaders)
	database.InsertResponse(db, requestID, 200, string(responseHeadersJSON), responseBody)

	go generateYAMLAutomatically(db, c.Request.URL.Path, c.Request.Method)

	c.Header("Content-Type", "application/json")
	c.JSON(200, gin.H{
		"message": "Simulated response",
		"status":  "ok",
		"path":    c.Request.URL.Path,
		"method":  c.Request.Method,
	})
}
