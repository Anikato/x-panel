package captcha

import (
	"github.com/mojocn/base64Captcha"
)

var store = base64Captcha.DefaultMemStore

func CreateCaptcha() (string, string, error) {
	driver := base64Captcha.NewDriverDigit(80, 240, 6, 0.7, 80)
	c := base64Captcha.NewCaptcha(driver, store)
	id, b64s, _, err := c.Generate()
	return id, b64s, err
}

func VerifyCaptcha(id, code string) bool {
	return store.Verify(id, code, true)
}
