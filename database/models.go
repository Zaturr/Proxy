package database

import (
	"time"
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
