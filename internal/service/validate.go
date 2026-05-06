package service

import (
	"regexp"
	"unicode"

	"github.com/tianlu1990s/gim/internal/model"
	"github.com/tianlu1990s/gim/pkg/errcode"
)

// 输入校验正则。userIDRegex 要求字母开头、4-32位、仅字母数字下划线，
// 与前端注册表单的校验规则保持一致。
var (
	userIDRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{3,31}$`)
	phoneRegex  = regexp.MustCompile(`^1[3-9]\d{9}$`)
	emailRegex  = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

// validateRegisterReq 校验注册请求，在 Service 层做业务校验（Gin binding 仅做基础格式校验）。
// 两层校验的原因：binding 校验通用格式（非空、长度），业务校验验证语义（格式正确性、密码强度）。
func validateRegisterReq(req *model.RegisterReq) error {
	if !userIDRegex.MatchString(req.UserID) {
		return errcode.ErrInvalidParam.WithDetail("userId 格式错误：4-32位，字母开头，仅字母数字下划线")
	}
	if !isStrongPassword(req.Password) {
		return errcode.ErrInvalidParam.WithDetail("密码强度不足：8-64位，须含大小写字母和数字")
	}
	if req.Phone != "" && !phoneRegex.MatchString(req.Phone) {
		return errcode.ErrInvalidParam.WithDetail("手机号格式错误")
	}
	if req.Email != "" && !emailRegex.MatchString(req.Email) {
		return errcode.ErrInvalidParam.WithDetail("邮箱格式错误")
	}
	return nil
}

// isStrongPassword 检查密码强度：8-64位，至少包含大写字母、小写字母和数字各一个。
func isStrongPassword(pwd string) bool {
	if len(pwd) < 8 || len(pwd) > 64 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range pwd {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// isDigit 判断字符串是否全为数字字符。
func isDigit(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return len(s) > 0
}
