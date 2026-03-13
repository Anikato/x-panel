package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"

	"xpanel/app/service"

	"github.com/gin-gonic/gin"
)

func NodeProxy() gin.HandlerFunc {
	nodeService := service.NewINodeService()
	return func(c *gin.Context) {
		nodeIDStr := c.GetHeader("X-Node-ID")
		if nodeIDStr == "" || nodeIDStr == "0" {
			c.Next()
			return
		}

		nodeID, err := strconv.ParseUint(nodeIDStr, 10, 64)
		if err != nil || nodeID == 0 {
			c.Next()
			return
		}

		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}

		data, statusCode, err := nodeService.ProxyRequest(uint(nodeID), c.Request.Method, c.Request.URL.Path, bytes.NewReader(bodyBytes))
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "proxy error: " + err.Error()})
			c.Abort()
			return
		}

		c.Data(statusCode, "application/json", data)
		c.Abort()
	}
}
