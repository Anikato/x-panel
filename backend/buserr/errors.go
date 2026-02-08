package buserr

import (
	"xpanel/i18n"
)

// BusinessError 业务错误，支持 i18n
type BusinessError struct {
	Msg    string
	Detail interface{}
	Map    map[string]interface{}
	Err    error
}

func (e BusinessError) Error() string {
	content := ""
	if e.Detail != nil {
		content = i18n.GetErrMsg(e.Msg, map[string]interface{}{"detail": e.Detail})
	} else if e.Map != nil {
		content = i18n.GetErrMsg(e.Msg, e.Map)
	} else {
		content = i18n.GetErrMsg(e.Msg, nil)
	}
	if content == "" {
		if e.Err != nil {
			return e.Err.Error()
		}
		return e.Msg
	}
	return content
}

// New 创建业务错误
func New(key string) BusinessError {
	return BusinessError{Msg: key}
}

// WithErr 创建带原始错误的业务错误
func WithErr(key string, err error) BusinessError {
	paramMap := map[string]interface{}{}
	if err != nil {
		paramMap["err"] = err.Error()
	}
	return BusinessError{
		Msg: key,
		Map: paramMap,
		Err: err,
	}
}

// WithDetail 创建带详情的业务错误
func WithDetail(key string, detail interface{}, err error) BusinessError {
	return BusinessError{
		Msg:    key,
		Detail: detail,
		Err:    err,
	}
}

// WithMap 创建带参数映射的业务错误
func WithMap(key string, maps map[string]interface{}, err error) BusinessError {
	return BusinessError{
		Msg: key,
		Map: maps,
		Err: err,
	}
}

// WithName 创建带 name 参数的业务错误
func WithName(key string, name string) BusinessError {
	paramMap := map[string]interface{}{}
	if name != "" {
		paramMap["name"] = name
	}
	return BusinessError{
		Msg: key,
		Map: paramMap,
	}
}
