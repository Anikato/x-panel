package helper

import (
	"net/http"

	"xpanel/app/dto"
	"xpanel/buserr"
	"xpanel/constant"
	"xpanel/i18n"

	"github.com/gin-gonic/gin"
)

// SuccessWithData 成功响应（带数据）
func SuccessWithData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// SuccessWithMsg 成功响应（带消息）
func SuccessWithMsg(c *gin.Context, msgKey string) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: i18n.GetMsgByKey(msgKey),
	})
}

// SuccessWithOutData 成功响应（无数据）
func SuccessWithOutData(c *gin.Context) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "success",
	})
}

// ErrorWithDetail 错误响应（带详情）
func ErrorWithDetail(c *gin.Context, code int, msg string) {
	c.JSON(code, dto.Response{
		Code:    code,
		Message: msg,
	})
}

// HandleError 统一处理 Service 层返回的错误
func HandleError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	switch e := err.(type) {
	case buserr.BusinessError:
		c.JSON(http.StatusOK, dto.Response{
			Code:    http.StatusInternalServerError,
			Message: e.Error(),
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.Response{
			Code:    http.StatusInternalServerError,
			Message: i18n.GetMsgWithMap(constant.ErrInternalServer, map[string]interface{}{"detail": err.Error()}),
		})
	}
}

// CheckBindAndValidate 绑定 JSON 参数并校验
func CheckBindAndValidate(req interface{}, c *gin.Context) error {
	if err := c.ShouldBindJSON(req); err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, err.Error(), err)
	}
	return nil
}

// CheckBindAndValidateQuery 绑定 Query 参数并校验
func CheckBindAndValidateQuery(req interface{}, c *gin.Context) error {
	if err := c.ShouldBindQuery(req); err != nil {
		return buserr.WithDetail(constant.ErrInvalidParams, err.Error(), err)
	}
	return nil
}

// GetClientIP 获取客户端 IP
func GetClientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		ip = c.Request.RemoteAddr
	}
	return ip
}

// GetUserAgent 获取 User-Agent
func GetUserAgent(c *gin.Context) string {
	return c.Request.UserAgent()
}

// GetToken 从请求头获取 Token
func GetToken(c *gin.Context) string {
	token := c.GetHeader(constant.JWTHeaderKey)
	if len(token) > len(constant.JWTTokenPrefix) {
		return token[len(constant.JWTTokenPrefix):]
	}
	return ""
}

// SuccessWithPage 分页成功响应
func SuccessWithPage(c *gin.Context, total int64, items interface{}) {
	c.JSON(http.StatusOK, dto.Response{
		Code:    0,
		Message: "success",
		Data: dto.PageResult{
			Total: total,
			Items: items,
		},
	})
}
