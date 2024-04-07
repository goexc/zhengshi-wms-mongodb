package validatorx

import (
	"fmt"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"reflect"
	"regexp"
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

	// 注册自定义校验规则
	Validator.RegisterValidation("mobile", MobileValidation)
	// 注册自定义错误
	Validator.RegisterTranslation("mobile", Trans, func(ut ut.Translator) error {
		return ut.Add("mobile", "{0}格式错误", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("mobile", fe.Field())
		return t
	})

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

// 手机号码校验
func MobileValidation(fl validator.FieldLevel) bool {
	verificationRole := `^1[345789]\d{9}$`
	field := fl.Field()

	switch field.Kind() {
	case reflect.String:
		re, err := regexp.Compile(verificationRole)
		if err != nil {
			fmt.Println("规则校验存在错误：", err.Error())
			return false
		}

		fmt.Println("规则校验结果：", re.MatchString(field.String()))

		return re.MatchString(field.String())
	default:
		return false
	}
}
