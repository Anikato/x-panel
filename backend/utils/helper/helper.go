package helper

import (
	"errors"
	"net/http"
	"strconv"

	"xpanel/app/dto"
	"xpanel/i18n"
	"github.com/gin-gonic/gin"
)

// ErrorWithDetail 返回带详细错误信息的响应
func ErrorWithDetail(ctx *gin.Context, code int, msgKey string) {
	res := dto.Response{
		Code:    code,
		Message: i18n.GetMsgByKey(msgKey),
	}
	ctx.JSON(http.StatusOK, res)
	ctx.Abort()
}

// InternalServer 返回内部服务器错误
func InternalServer(ctx *gin.Context, err error) {
	ErrorWithDetail(ctx, http.StatusInternalServerError, "ErrInternalServer")
}

// BadRequest 返回请求参数错误
func BadRequest(ctx *gin.Context, err error) {
	ErrorWithDetail(ctx, http.StatusBadRequest, "ErrInvalidParams")
}

// BadAuth 返回认证错误
func BadAuth(ctx *gin.Context, msgKey string) {
	ErrorWithDetail(ctx, http.StatusUnauthorized, msgKey)
}

// SuccessWithData 返回成功响应和数据
func SuccessWithData(ctx *gin.Context, data interface{}) {
	if data == nil {
		data = gin.H{}
	}
	res := dto.Response{
		Code: http.StatusOK,
		Data: data,
	}
	ctx.JSON(http.StatusOK, res)
}

// SuccessWithMsg 返回成功响应和消息
func SuccessWithMsg(ctx *gin.Context, msgKey string) {
	res := dto.Response{
		Code:    http.StatusOK,
		Message: i18n.GetMsgByKey(msgKey),
	}
	ctx.JSON(http.StatusOK, res)
}

// Success 返回成功响应
func Success(ctx *gin.Context) {
	res := dto.Response{
		Code:    http.StatusOK,
		Message: "success",
	}
	ctx.JSON(http.StatusOK, res)
}

// CheckBindAndValidate 绑定并验证请求参数
func CheckBindAndValidate(req interface{}, c *gin.Context) error {
	if err := c.ShouldBindJSON(req); err != nil {
		ErrorWithDetail(c, http.StatusBadRequest, "ErrInvalidParams")
		return err
	}
	// TODO: Add validation when validator is set up
	return nil
}

// GetParamID 从路径参数中获取 ID
func GetParamID(c *gin.Context) (uint, error) {
	idParam, ok := c.Params.Get("id")
	if !ok {
		return 0, errors.New("error id in path")
	}
	intNum, _ := strconv.Atoi(idParam)
	return uint(intNum), nil
}