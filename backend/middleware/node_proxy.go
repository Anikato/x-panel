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

		// 仅在确认需要代理时才读取 body
		// 对 multipart 上传请求，直接透传 body 流，不缓冲到内存
		var bodyReader io.Reader
		if c.Request.Body != nil {
			bodyReader = c.Request.Body
		} else {
			bodyReader = bytes.NewReader(nil)
		}

		data, statusCode, err := nodeService.ProxyRequest(uint(nodeID), c.Request.Method, c.Request.URL.Path, bodyReader)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"code": 502, "message": "proxy error: " + err.Error()})
			c.Abort()
			return
		}

		c.Data(statusCode, "application/json", data)
		c.Abort()
	}
}
