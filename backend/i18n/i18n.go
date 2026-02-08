package i18n

import (
	"embed"
	"fmt"
	"strings"

	"xpanel/global"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

//go:embed lang/*
var fs embed.FS
var bundle *i18n.Bundle

// Init 初始化 i18n 模块（当前仅支持中文）
func Init() {
	bundle = i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	if _, err := bundle.LoadMessageFileFS(fs, "lang/zh.yaml"); err != nil {
		fmt.Printf("[i18n] load zh.yaml failed: %v\n", err)
	}

	global.I18n = i18n.NewLocalizer(bundle, "zh")
	global.LOG.Info("i18n initialized (zh)")
}

// GetMsgWithMap 获取 i18n 消息（带参数映射）
func GetMsgWithMap(key string, maps map[string]interface{}) string {
	if global.I18n == nil {
		return key
	}
	var content string
	if maps == nil {
		content, _ = global.I18n.Localize(&i18n.LocalizeConfig{
			MessageID: key,
		})
	} else {
		content, _ = global.I18n.Localize(&i18n.LocalizeConfig{
			MessageID:    key,
			TemplateData: maps,
		})
	}
	content = strings.ReplaceAll(content, ": <no value>", "")
	if content == "" {
		return key
	}
	return content
}

// GetErrMsg 获取错误消息
func GetErrMsg(key string, maps map[string]interface{}) string {
	if global.I18n == nil {
		return key
	}
	var content string
	if maps == nil {
		content, _ = global.I18n.Localize(&i18n.LocalizeConfig{
			MessageID: key,
		})
	} else {
		content, _ = global.I18n.Localize(&i18n.LocalizeConfig{
			MessageID:    key,
			TemplateData: maps,
		})
	}
	return content
}

// GetMsgByKey 获取 i18n 消息
func GetMsgByKey(key string) string {
	if global.I18n == nil {
		return key
	}
	content, _ := global.I18n.Localize(&i18n.LocalizeConfig{
		MessageID: key,
	})
	if content != "" {
		return content
	}
	return key
}
