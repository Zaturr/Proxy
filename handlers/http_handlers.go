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
	infra "proxy/database/infraestructure"
	"proxy/handlers/port"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ProxyConfig contiene la configuración del proxy
type ProxyConfig struct {
	HTTPSPorts  map[string]bool
	DefaultPort string
}

// PortMapper maneja el mapeo de paths a puertos
type PortMapper struct {
	config *ProxyConfig
}

// RequestLogger maneja el logging de requests
type RequestLogger struct {
	db *sql.DB
}

// ResponseLogger maneja el logging de responses
type ResponseLogger struct {
	db *sql.DB
}

// YAMLGenerator maneja la generación de YAML
type YAMLGenerator struct {
	db *sql.DB
}

// ProxyService coordina todos los servicios del proxy
type ProxyService struct {
	portMapper     *PortMapper
	requestLogger  *RequestLogger
	responseLogger *ResponseLogger
	yamlGenerator  *YAMLGenerator
	config         *ProxyConfig
}

// ProxyError representa errores del proxy
type ProxyError struct {
	Type    string
	Message string
	Err     error
}

func (e *ProxyError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewProxyService crea una nueva instancia del servicio de proxy
func NewProxyService(db *sql.DB) *ProxyService {
	config := &ProxyConfig{
		HTTPSPorts: map[string]bool{
			"8443": true,
			"443":  true,
		},
		DefaultPort: "8080",
	}

	return &ProxyService{
		portMapper:     NewPortMapper(config),
		requestLogger:  NewRequestLogger(db),
		responseLogger: NewResponseLogger(db),
		yamlGenerator:  NewYAMLGenerator(db),
		config:         config,
	}
}

// Constructores simplificados
func NewPortMapper(config *ProxyConfig) *PortMapper { return &PortMapper{config: config} }
func NewRequestLogger(db *sql.DB) *RequestLogger    { return &RequestLogger{db: db} }
func NewResponseLogger(db *sql.DB) *ResponseLogger  { return &ResponseLogger{db: db} }
func NewYAMLGenerator(db *sql.DB) *YAMLGenerator    { return &YAMLGenerator{db: db} }

// GetTargetPort mapea un path a un puerto específico
func (pm *PortMapper) GetTargetPort(path string) string {
	return port.GetTargetPortByPath(path, pm.config.DefaultPort)
}

// GetTargetProtocol determina el protocolo basado en el puerto
func (pm *PortMapper) GetTargetProtocol(port string) string {
	if pm.config.HTTPSPorts[port] {
		return "https"
	}
	return "http"
}

// LogRequest registra un request en la base de datos
func (rl *RequestLogger) LogRequest(method, url, headers, body string, port int) (int64, error) {
	requestID, err := infra.InsertRequest(rl.db, method, url, headers, body, port)
	if err != nil {
		return 0, &ProxyError{Type: "DatabaseInsert", Message: "Failed to insert request", Err: err}
	}
	return requestID, nil
}

// LogResponse registra una response en la base de datos
func (rl *ResponseLogger) LogResponse(requestID int64, statusCode int, headers, body string, port int) error {
	if err := infra.InsertResponse(rl.db, requestID, statusCode, headers, body, port); err != nil {
		return &ProxyError{Type: "DatabaseInsert", Message: "Failed to insert response", Err: err}
	}
	return nil
}

// GenerateYAMLAsync genera YAML de forma asíncrona
func (yg *YAMLGenerator) GenerateYAMLAsync(path, method string) {
	go func() {
		_, err := yaml.GenerateYAMLFromDB(yg.db, path, method)
		if err != nil {
			fmt.Printf("Error generating YAML automatically: %v\n", err)
		}
	}()
}

// MockingbirdProxy maneja las requests del proxy usando el servicio refactorizado
func MockingbirdProxy(db *sql.DB, c *gin.Context) {
	proxyService := NewProxyService(db)
	proxyService.HandleProxyRequest(c)
}

// HandleProxyRequest maneja una request del proxy
func (ps *ProxyService) HandleProxyRequest(c *gin.Context) {
	path := c.Param("path")

	// Obtener puerto y protocolo usando el servicio
	targetPort := ps.portMapper.GetTargetPort(path)
	protocol := ps.portMapper.GetTargetProtocol(targetPort)
	targetURL := protocol + "://localhost:" + targetPort

	fmt.Printf("Path: %s, Target Port: %s, Protocol: %s, Target URL: %s\n", path, targetPort, protocol, targetURL)

	remote, err := url.Parse(targetURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse target URL"})
		return
	}

	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			ps.handleRequestRewrite(r, c, targetPort, remote)
		},
		ModifyResponse: func(r *http.Response) error {
			return ps.handleResponseModify(r, c, targetPort)
		},
	}

	proxy.ServeHTTP(c.Writer, c.Request)
}

// handleRequestRewrite maneja la lógica de rewrite del request
func (ps *ProxyService) handleRequestRewrite(r *httputil.ProxyRequest, c *gin.Context, targetPort string, remote *url.URL) {
	r.SetURL(remote)
	r.Out.Host = r.In.Host

	body, _ := io.ReadAll(r.In.Body)
	headersJSON, _ := json.Marshal(r.In.Header)
	portInt, _ := strconv.Atoi(targetPort)

	if requestID, err := ps.requestLogger.LogRequest(r.In.Method, r.In.URL.String(), string(headersJSON), string(body), portInt); err != nil {
		fmt.Printf("Error logging request: %v\n", err)
	} else {
		c.Set("requestID", requestID)
	}

	ps.yamlGenerator.GenerateYAMLAsync(r.In.URL.Path, r.In.Method)
	fmt.Printf("Request body: %s\n", body)
	r.Out.Body = io.NopCloser(bytes.NewBuffer(body))
}

// handleResponseModify maneja la lógica de modificación de response
func (ps *ProxyService) handleResponseModify(r *http.Response, c *gin.Context, targetPort string) error {
	body, _ := io.ReadAll(r.Body)

	if requestID, exists := c.Get("requestID"); exists {
		headersJSON, _ := json.Marshal(r.Header)
		portInt, _ := strconv.Atoi(targetPort)
		if err := ps.responseLogger.LogResponse(requestID.(int64), r.StatusCode, string(headersJSON), string(body), portInt); err != nil {
			fmt.Printf("Error logging response: %v\n", err)
		}
	}

	fmt.Printf("Response body: %s\n", body)
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	return nil
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

// SimulateResponse simula una respuesta cuando no hay servidor de destino
func SimulateResponse(db *sql.DB, c *gin.Context) {
	proxyService := NewProxyService(db)
	proxyService.HandleSimulatedResponse(c)
}

// HandleSimulatedResponse maneja la simulación de respuesta
func (ps *ProxyService) HandleSimulatedResponse(c *gin.Context) {
	headersJSON, _ := json.Marshal(c.Request.Header)
	body, _ := io.ReadAll(c.Request.Body)
	path := c.Request.URL.Path
	targetPort := ps.portMapper.GetTargetPort(path)
	portInt, _ := strconv.Atoi(targetPort)

	requestID, _ := ps.requestLogger.LogRequest(c.Request.Method, c.Request.URL.String(), string(headersJSON), string(body), portInt)

	responseBody := `{"message": "Simulated response", "status": "ok"}`
	responseHeadersJSON, _ := json.Marshal(map[string]string{"Content-Type": "application/json"})
	ps.responseLogger.LogResponse(requestID, 200, string(responseHeadersJSON), responseBody, portInt)
	ps.yamlGenerator.GenerateYAMLAsync(c.Request.URL.Path, c.Request.Method)

	c.Header("Content-Type", "application/json")
	c.JSON(200, gin.H{"message": "Simulated response", "status": "ok", "path": c.Request.URL.Path, "method": c.Request.Method})
}

// StatsHandler maneja las estadísticas de requests
func StatsHandler(db *sql.DB, c *gin.Context) {
	if stats, err := yaml.GetRequestStats(db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
	} else {
		c.JSON(http.StatusOK, stats)
	}
}

// CountHandler maneja los conteos específicos
func CountHandler(db *sql.DB, c *gin.Context) {
	countType := c.Query("type")

	handlers := map[string]func() (interface{}, error){
		"approved":    func() (interface{}, error) { return yaml.CountApprovedRequests(db) },
		"rejected":    func() (interface{}, error) { return yaml.CountRejectedRequests(db) },
		"yaml_paths":  func() (interface{}, error) { return yaml.GetUniquePathsForYAML(db) },
		"error_paths": func() (interface{}, error) { return yaml.GetErrorPathsForChaosInjection(db) },
	}

	responseKeys := map[string]string{
		"approved": "approved_requests", "rejected": "rejected_requests",
		"yaml_paths": "yaml_paths", "error_paths": "error_paths",
	}

	if handler, exists := handlers[countType]; exists {
		if data, err := handler(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get " + countType})
		} else {
			c.JSON(http.StatusOK, gin.H{responseKeys[countType]: data})
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid count type. Use: approved, rejected, yaml_paths, error_paths"})
	}
}
