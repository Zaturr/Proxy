package database

import (
	"context"
	"database/sql"
	"fmt"
	"proxy/database/internal"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

type ProxyRequest struct {
	ID        int64     `json:"id" db:"id"`
	Method    string    `json:"method" db:"method"`
	URL       string    `json:"url" db:"url"`
	Headers   string    `json:"headers" db:"headers"`
	Body      string    `json:"body" db:"body"`
	Timestamp time.Time `json:"timestamp" db:"timestamp"`
}

type ProxyResponse struct {
	ID         int64     `json:"id" db:"id"`
	RequestID  int64     `json:"request_id" db:"request_id"`
	StatusCode int       `json:"status_code" db:"status_code"`
	Headers    string    `json:"headers" db:"headers"`
	Body       string    `json:"body" db:"body"`
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

type ProxyTransaction struct {
	Request  ProxyRequest  `json:"request"`
	Response ProxyResponse `json:"response"`
}

type BestMatch struct {
	Endpoint   string            `json:"endpoint"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code"`
	Score      int               `json:"score"`
	Method     string            `json:"method"`
	URL        string            `json:"url"`
	Port       int               `json:"port"`
}

type SearchCriteria struct {
	URL      string            `json:"url"`
	Endpoint string            `json:"endpoint"`
	Body     string            `json:"body"`
	Headers  map[string]string `json:"headers"`
	Method   string            `json:"method"`
}

type MockingbirdConfig struct {
	HTTP HTTPConfig `yaml:"http"`
}

type HTTPConfig struct {
	Servers []ServerConfig `yaml:"servers"`
}

type ServerConfig struct {
	Listen         int              `yaml:"listen"`
	Logger         bool             `yaml:"logger"`
	Name           string           `yaml:"name"`
	LoggerPath     string           `yaml:"logger_path"`
	Version        string           `yaml:"version"`
	Location       []LocationConfig `yaml:"location"`
	ChaosInjection *ChaosInjection  `yaml:"chaos_injection,omitempty"`
}

type LocationConfig struct {
	Path       string   `yaml:"path"`
	Method     string   `yaml:"method"`
	Response   string   `yaml:"response"`
	StatusCode int      `yaml:"status_code"`
	Headers    *Headers `yaml:"headers"`
	Schema     string   `yaml:"schema,omitempty"`
}
type Headers struct {
	ContentType string `yaml:"content_type,omitempty"`
}
type ChaosInjection struct {
	Probability float64 `yaml:"probability"`
	StatusCode  int     `yaml:"status_code"`
}

// Mockingbird Database Structures

// Mockdata representa los datos de una transacción mock
type Mockdata struct {
	UUID               string    `json:"uuid" db:"uuid"`
	RequestHeaders     string    `json:"request_headers" db:"request_headers"`
	RequestMethod      string    `json:"request_method" db:"request_method"`
	RequestEndpoint    string    `json:"request_endpoint" db:"request_endpoint"`
	RequestBody        string    `json:"request_body" db:"request_body"`
	ResponseHeaders    string    `json:"response_headers" db:"response_headers"`
	ResponseBody       string    `json:"response_body" db:"response_body"`
	ResponseStatusCode int       `json:"response_status_code" db:"response_status_code"`
	Timestamp          time.Time `json:"timestamp" db:"timestamp"`
    Port               int       `json:"port" db:"port"`
}

// WorkerConfig configuración del worker
type WorkerConfig struct {
	MaxWorkers    int           `json:"max_workers"`    // Número máximo de workers concurrentes
	QueueSize     int           `json:"queue_size"`     // Tamaño de la cola de trabajos
	Timeout       time.Duration `json:"timeout"`        // Timeout para operaciones
	RetryAttempts int           `json:"retry_attempts"` // Número de reintentos en caso de error
}

// Worker maneja las operaciones de inserción síncronas y asíncronas
type Worker struct {
	DB          *sql.DB
	Config      WorkerConfig
	JobQueue    chan *Mockdata
	ResultQueue chan error
	Ctx         context.Context
	Cancel      context.CancelFunc
	WaitGroup   sync.WaitGroup
	TimeStop    sync.RWMutex
	Running     bool
}

// BatchConfig configuración del sistema de batch
type BatchConfig struct {
	BatchSize     int           `json:"batch_size"`      // Tamaño del batch (default: 20)
	FlushInterval time.Duration `json:"flush_interval"`  // Intervalo para flush automático
	MaxQueueSize  int           `json:"max_queue_size"`  // Tamaño máximo de la cola de entrada
	MaxBatchQueue int           `json:"max_batch_queue"` // Tamaño máximo de la cola de batches
	MaxWorkers    int           `json:"max_workers"`     // Número de workers para procesar batches
	Timeout       time.Duration `json:"timeout"`         // Timeout para operaciones
	RetryAttempts int           `json:"retry_attempts"`  // Número de reintentos
	EnableMetrics bool          `json:"enable_metrics"`  // Habilitar métricas
}

// Batch representa un lote de operaciones
type Batch struct {
	ID         string      `json:"id"`
	Operations []*Mockdata `json:"operations"`
	CreatedAt  time.Time   `json:"created_at"`
	Size       int         `json:"size"`
}

// QueueManager maneja todas las colas del sistema
type QueueManager struct {
	InputQueue  chan *Mockdata
	BatchQueue  chan *Batch
	ResultQueue chan error
	Ctx         context.Context
	Cancel      context.CancelFunc
	WaitGroup   sync.WaitGroup
	Running     bool
	Mutex       sync.RWMutex
}

// BatchManager maneja el sistema de batch con alta concurrencia
type BatchManager struct {
	DB        *sql.DB
	Config    BatchConfig
	QueueMgr  *QueueManager
	WaitGroup sync.WaitGroup
	Running   bool
	Mutex     sync.RWMutex

	TotalProcessed int64
	TotalBatches   int64
	TotalErrors    int64
	CurrentBatch   *Batch
	BatchMutex     sync.Mutex
	LastFlush      time.Time
	FlushTicker    *time.Ticker
}

// InsertOperation inserta una nueva operación en la base de datos
func InsertOperation(db *sql.DB, operation *Mockdata) error {
	internalOp := &internal.Mockdata{
		UUID:               operation.UUID,
		RequestHeaders:     operation.RequestHeaders,
		RequestMethod:      operation.RequestMethod,
		RequestEndpoint:    operation.RequestEndpoint,
		RequestBody:        operation.RequestBody,
		ResponseHeaders:    operation.ResponseHeaders,
		ResponseBody:       operation.ResponseBody,
		ResponseStatusCode: operation.ResponseStatusCode,
		Timestamp:          operation.Timestamp,
        Port:               operation.Port,
	}
	return internal.InsertOperation(db, internalOp)
}

// UpdateOperationResponse actualiza solo la respuesta de una operación
func UpdateOperationResponse(db *sql.DB, uuid string, responseHeaders, responseBody string, statusCode int) error {
	return internal.UpdateOperationResponse(db, uuid, responseHeaders, responseBody, statusCode)
}

// ConvertToMockdata convierte ProxyRequest y ProxyResponse a Mockdata
func ConvertToMockdata(request *ProxyRequest, response *ProxyResponse) *Mockdata {
	return &Mockdata{
		UUID:               fmt.Sprintf("req_%d", request.ID),
		RequestHeaders:     request.Headers,
		RequestMethod:      request.Method,
		RequestEndpoint:    request.URL,
		RequestBody:        request.Body,
		ResponseHeaders:    response.Headers,
		ResponseBody:       response.Body,
		ResponseStatusCode: response.StatusCode,
		Timestamp:          request.Timestamp,
	}
}

// ConvertFromMockdata convierte Mockdata a ProxyRequest y ProxyResponse
func ConvertFromMockdata(mockdata *Mockdata) (*ProxyRequest, *ProxyResponse) {
	request := &ProxyRequest{
		Method:    mockdata.RequestMethod,
		URL:       mockdata.RequestEndpoint,
		Headers:   mockdata.RequestHeaders,
		Body:      mockdata.RequestBody,
		Timestamp: mockdata.Timestamp,
	}

	response := &ProxyResponse{
		StatusCode: mockdata.ResponseStatusCode,
		Headers:    mockdata.ResponseHeaders,
		Body:       mockdata.ResponseBody,
		Timestamp:  mockdata.Timestamp,
	}

	return request, response
}

// InsertTransactionUnified inserta una transacción completa en la tabla unificada
func InsertTransactionUnified(db *sql.DB, request *ProxyRequest, response *ProxyResponse) error {
	mockdata := ConvertToMockdata(request, response)
	return InsertOperation(db, mockdata)
}

// GetTransactionUnified obtiene una transacción desde la tabla unificada
func GetTransactionUnified(db *sql.DB, uuid string) (*ProxyTransaction, error) {
	internalOp, err := internal.GetOperationByUUID(db, uuid)
	if err != nil {
		return nil, err
	}

	mockdata := &Mockdata{
		UUID:               internalOp.UUID,
		RequestHeaders:     internalOp.RequestHeaders,
		RequestMethod:      internalOp.RequestMethod,
		RequestEndpoint:    internalOp.RequestEndpoint,
		RequestBody:        internalOp.RequestBody,
		ResponseHeaders:    internalOp.ResponseHeaders,
		ResponseBody:       internalOp.ResponseBody,
		ResponseStatusCode: internalOp.ResponseStatusCode,
		Timestamp:          internalOp.Timestamp,
        Port:               internalOp.Port,
	}

	request, response := ConvertFromMockdata(mockdata)
	return &ProxyTransaction{
		Request:  *request,
		Response: *response,
	}, nil
}

// GetAllTransactionsUnified obtiene todas las transacciones desde la tabla unificada
func GetAllTransactionsUnified(db *sql.DB, limit, offset int) ([]ProxyTransaction, error) {
	internalOps, err := internal.GetAllOperations(db, limit, offset)
	if err != nil {
		return nil, err
	}

	var transactions []ProxyTransaction
	for _, internalOp := range internalOps {
		mockdata := &Mockdata{
			UUID:               internalOp.UUID,
			RequestHeaders:     internalOp.RequestHeaders,
			RequestMethod:      internalOp.RequestMethod,
			RequestEndpoint:    internalOp.RequestEndpoint,
			RequestBody:        internalOp.RequestBody,
			ResponseHeaders:    internalOp.ResponseHeaders,
			ResponseBody:       internalOp.ResponseBody,
			ResponseStatusCode: internalOp.ResponseStatusCode,
			Timestamp:          internalOp.Timestamp,
            Port:               internalOp.Port,
		}

		request, response := ConvertFromMockdata(mockdata)
		transactions = append(transactions, ProxyTransaction{
			Request:  *request,
			Response: *response,
		})
	}

	return transactions, nil
}
