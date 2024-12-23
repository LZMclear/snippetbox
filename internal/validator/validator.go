package validator

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Validator 存储表单数据验证出现的错误
type Validator struct {
	FieldErrors    map[string]string
	NonFieldErrors []string //存储与特定表单字段无关的错误
}

// Valid 如果FieldErrors没有错误返回true
func (v *Validator) Valid() bool {
	return len(v.FieldErrors) == 0 && len(v.NonFieldErrors) == 0
}

// AddFieldError 向FieldErrors添加错误信息
func (v *Validator) AddFieldError(key string, message string) {
	if v.FieldErrors == nil {
		v.FieldErrors = make(map[string]string)
	}
	//先判断有没有这个字段
	if _, s := v.FieldErrors[key]; !s {
		v.FieldErrors[key] = message
	}
}

// NotBlank returns true if a value is not an empty string.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// CheckField adds an error message to the FieldErrors map only if a
// validation check is not 'ok'.
func (v *Validator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

// MaxChars returns true if a value contains no more than n characters.
func MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// PermittedValue returns true if a value is in a list of permitted integers.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	for i := range permittedValues {
		if value == permittedValues[i] {
			return true
		}
	}
	return false
}

// EmailRX 使用regexp.MustCompile()解析一个正则表达式检查电子邮件地址的格式 返回一个已编译正则表达式的regexp.Regexp类型的指针，或在发生错误时panic
// 在启动时解析此模式一次之后存储编译后的regexp.Regexp变量比每次需要时重新解析模式性能更好
var EmailRX = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// MinChars returns true if a value contains at least n characters.
func MinChars(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// Matches 返回真如果一个值匹配编译后的正则表达式
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// AddNonFieldError 向添加NonFieldErrors添加新错误
func (v *Validator) AddNonFieldError(message string) {
	v.NonFieldErrors = append(v.NonFieldErrors, message)
}
