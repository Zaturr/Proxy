package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	build "proxy/Build"
	"proxy/database"
	"strings"

	"github.com/gin-gonic/gin"
)

func proxy(c *gin.Context) {
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

func MockingbirdProxy(c *gin.Context) {
	path := c.Param("path")

	// Mapear rutas a puertos de Mockingbird
	var targetPort string
	switch {
	case strings.HasPrefix(path, "/jsonplaceholder"):
		targetPort = "8080"
	case strings.Contains(path, "/callback"):
		targetPort = "8080"
	case strings.Contains(path, "/auth"):
		targetPort = "8086"
	case strings.Contains(path, "/api"):
		targetPort = "8081"
	default:
		targetPort = "8080" // Default
	}

	targetURL := "http://localhost" + ":" + targetPort
	fmt.Printf("Path: %s, Target Port: %s, Target URL: %s\n", path, targetPort, targetURL)
	remote, _ := url.Parse(targetURL)

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(remote)
			r.Out.Host = r.In.Host // if desired
			body, err := io.ReadAll(r.In.Body)

			if err != nil {
				fmt.Errorf("Error %s", err.Error())
			}
			// Guardar request en base de datos
			headersJSON, _ := json.Marshal(r.In.Header)
			requestID, _ := database.InsertRequest(db, r.In.Method, r.In.URL.String(), string(headersJSON), string(body))
			c.Set("requestID", requestID)

			fmt.Println("%s", body)
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

			fmt.Println("%s", body)
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			return nil
		},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func searchHandler(c *gin.Context) {
	var criteria database.SearchCriteria

	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Realizar búsqueda jerárquica
	bestMatch, err := database.HierarchicalSearch(db, criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	// Si no se encontró ningún resultado, retornar 404
	if bestMatch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No matching records found"})
		return
	}

	// Retornar el mejor candidato encontrado
	c.JSON(http.StatusOK, bestMatch)
}

func searchConfigHandler(c *gin.Context) {
	var criteria database.SearchCriteria

	if err := c.ShouldBindJSON(&criteria); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
		return
	}

	// Realizar búsqueda jerárquica
	bestMatch, err := database.HierarchicalSearch(db, criteria)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Search failed"})
		return
	}

	// Si no se encontró ningún resultado, retornar 404
	if bestMatch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No matching records found"})
		return
	}

	// Convertir database.BestMatch a build.BestMatch
	buildBestMatch := &build.BestMatch{
		Endpoint:   bestMatch.Endpoint,
		Headers:    bestMatch.Headers,
		Body:       bestMatch.Body,
		StatusCode: bestMatch.StatusCode,
		Score:      bestMatch.Score,
		Method:     bestMatch.Method,
		URL:        bestMatch.URL,
	}

	// Generar configuración de Mockingbird
	config := build.GenerateMockingbirdConfig(buildBestMatch)

	// Retornar la configuración generada
	c.JSON(http.StatusOK, config)
}

var db *sql.DB

func main() {
	// Inicializar base de datos
	var err error
	db, err = database.InitDB("./database/proxy.db")
	if err != nil {
		fmt.Printf("Error initializing database: %v\n", err)
		return
	}
	defer db.Close()

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	})

	// Rutas para búsqueda (comentadas temporalmente)
	//r.POST("/search", searchHandler)
	//r.POST("/search/config", searchConfigHandler)

	// Rutas para Mockingbird
	r.Any("/*path", MockingbirdProxy)

	r.Run(":3000")
}
