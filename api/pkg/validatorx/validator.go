package validatorx

import (
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
)

var Validator *validator.Validate
var Trans ut.Translator

func init() {
	var locale = "zh"
	//创建通用翻译器
	uni := ut.New(zh.New(), en.New())
	Trans, _ = uni.GetTranslator(locale)

	//创建验证器
	Validator = validator.New()

	//翻译器注册到validator
	switch locale {
	case "zh":
		zhTranslations.RegisterDefaultTranslations(Validator, Trans)
		//使用field.Tag.Get("comment")注册一个获取tag的自定义方法
		Validator.RegisterTagNameFunc(func(field reflect.StructField) string {
			return field.Tag.Get("comment")
		})
	case "en":
		enTranslations.RegisterDefaultTranslations(Validator, Trans)
		Validator.RegisterTagNameFunc(func(field reflect.StructField) string {
			return field.Tag.Get("en_comment")
		})
	}
}
