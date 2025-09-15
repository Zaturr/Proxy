package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
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

	case strings.HasPrefix(path, "/callback"):
		targetPort = "8080"
	case strings.Contains(path, "/auth"):
		targetPort = "8086"
	default:
		targetPort = "8080" // Default
	}

	targetURL := "https://pruebas.sypago.net" + ":" + targetPort
	remote, _ := url.Parse(targetURL)

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(remote)
			r.Out.Host = r.In.Host // if desired
			body, err := io.ReadAll(r.In.Body)

			if err != nil {
				fmt.Errorf("Error %s", err.Error())
			}
			fmt.Println("%s", body)
			r.Out.Body = io.NopCloser(bytes.NewBuffer(body))
		},
		ModifyResponse: func(r *http.Response) error {
			body, err := io.ReadAll(r.Body)

			if err != nil {
				return err
			}
			fmt.Println("%s", body)

			r.Body = io.NopCloser(bytes.NewBuffer(body))
			return nil
		},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

func main() {
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

	// Rutas para Mockingbird
	r.Any("/*path", MockingbirdProxy)

	// Proxy externo
	//r.Any("/*proxyPath", proxy)

	fmt.Println(" Proxy server running on :3000")
	fmt.Println(" External proxy: http://localhost:3000/users?url=https://jsonplaceholder.typicode.com")
	fmt.Println(" Mockingbird routes:")
	fmt.Println(" - http://localhost:3000/mockingbird/jsonplaceholder/*")
	fmt.Println(" - http://localhost:3000/mockingbird/sypago/*")
	fmt.Println(" - http://localhost:3000/mockingbird/users/*")

	r.Run(":3000")
}
