package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/apps/tags", func(c *gin.Context) { c.String(200, "tags") })
	r.GET("/apps/:key", func(c *gin.Context) { c.String(200, "key="+c.Param("key")) })
	r.GET("/apps/detail", func(c *gin.Context) { c.String(200, "detail") })

	req := httptest.NewRequest(http.MethodGet, "/apps/detail?appId=1&version=1.0", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	fmt.Println("status", w.Code, "body", w.Body.String())
}
