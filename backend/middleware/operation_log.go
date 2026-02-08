package middleware

import (
	"bytes"
	"io"
	"strings"
	"time"

	"xpanel/app/model"
	"xpanel/app/repo"
	"xpanel/constant"
	"xpanel/global"

	"github.com/gin-gonic/gin"
)

// OperationLog 操作日志中间件
func OperationLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 仅记录写操作（POST/PUT/DELETE）
		method := c.Request.Method
		if method == "GET" || method == "OPTIONS" || method == "HEAD" {
			c.Next()
			return
		}

		// 读取请求体
		var body string
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err == nil {
				body = string(bodyBytes)
				// 恢复请求体供后续处理使用
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// 脱敏：移除密码等敏感字段
		body = maskSensitiveFields(body)

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// 获取路径信息
		path := c.Request.URL.Path
		group, source, action := parsePathInfo(path)

		// 确定操作状态
		status := constant.StatusSuccess
		message := ""
		if len(c.Errors) > 0 {
			status = constant.StatusFailed
			message = c.Errors.Last().Error()
		}

		// 记录耗时
		if message == "" {
			message = duration.String()
		}

		// 异步写入日志
		log := &model.OperationLog{
			Group:  group,
			Source: source,
			Action: action,
			IP:     c.ClientIP(),
			Path:   path,
			Method: method,
			Body:   body,
			Status: status,
			Message: message,
		}

		go func() {
			logRepo := repo.NewILogRepo()
			if err := logRepo.CreateOperationLog(log); err != nil {
				global.LOG.Errorf("Failed to save operation log: %v", err)
			}
		}()
	}
}

// parsePathInfo 从路径解析分组、来源和操作
// 路径格式：/api/v1/{group}/{action}
func parsePathInfo(path string) (group, source, action string) {
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/"), "/")
	if len(parts) >= 1 {
		group = parts[0]
	}
	if len(parts) >= 2 {
		action = parts[1]
	}
	source = group
	return
}

// maskSensitiveFields 脱敏敏感字段
func maskSensitiveFields(body string) string {
	sensitiveKeys := []string{"password", "newPassword", "oldPassword", "secret"}
	for _, key := range sensitiveKeys {
		// 匹配 "key":"value" 模式并替换 value 为 ***
		pattern := `"` + key + `"`
		idx := strings.Index(body, pattern)
		for idx != -1 {
			// 跳过 key 本身和后续的 ":"
			start := idx + len(pattern)
			rest := body[start:]
			// 查找冒号后的引号包裹的值
			colonIdx := strings.Index(rest, `:"`)
			if colonIdx != -1 && colonIdx < 3 {
				// 找到值的起始引号
				valueStart := start + colonIdx + 2
				// 找到值的结束引号
				valueEnd := strings.Index(body[valueStart:], `"`)
				if valueEnd != -1 {
					body = body[:valueStart] + "***" + body[valueStart+valueEnd:]
				}
			}
			// 继续搜索下一个匹配
			nextSearch := idx + len(pattern)
			if nextSearch >= len(body) {
				break
			}
			nextIdx := strings.Index(body[nextSearch:], pattern)
			if nextIdx == -1 {
				break
			}
			idx = nextSearch + nextIdx
		}
	}
	return body
}
