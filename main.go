package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"proxy/database"
	"proxy/handlers"

	"github.com/gin-gonic/gin"
)

func MockingbirdProxyWrapper(c *gin.Context) {
	handlers.MockingbirdProxy(db, c)
}

func SearchHandlerWrapper(c *gin.Context) {
	handlers.SearchHandler(db, c)
}

func SearchConfigHandlerWrapper(c *gin.Context) {
	handlers.SearchConfigHandler(db, c)
}

var db *sql.DB

func main() {
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

	r.Any("/*path", func(c *gin.Context) {
		path := c.Request.URL.Path

		if path == "/api/search" && c.Request.Method == "POST" {
			SearchHandlerWrapper(c)
			return
		}
		if path == "/api/search/config" && c.Request.Method == "POST" {
			SearchConfigHandlerWrapper(c)
			return
		}
		MockingbirdProxyWrapper(c)
	})

	r.Run(":3000")
}
