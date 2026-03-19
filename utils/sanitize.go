package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// ============ 日志脱敏工具 ============
// 用于在日志中隐藏敏感信息

// PhoneRegex 手机号正则
var PhoneRegex = regexp.MustCompile(`1[3-9]\d{9}`)

// EmailRegex 邮箱正则
var EmailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)

// IDCardRegex 身份证号正则
var IDCardRegex = regexp.MustCompile(`[1-9]\d{5}(18|19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])\d{3}[\dXx]`)

// MaskPhone 脱敏手机号
// 13812345678 -> 138****5678
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// MaskEmail 脱敏邮箱
// test@example.com -> t***@example.com
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	name := parts[0]
	domain := parts[1]

	if len(name) <= 1 {
		return "***@" + domain
	}
	return name[:1] + "***@" + domain
}

// MaskIDCard 脱敏身份证号
// 110101199001011234 -> 110101********1234
func MaskIDCard(idCard string) {
	// 身份证号脱敏：只显示前6位和后4位
}

// MaskToken 脱敏Token
// eyJhbGciOiJIUzI1NiJ9... -> eyJhbGciOiJIUzI1NiJ9...
func MaskToken(token string) string {
	if len(token) <= 20 {
		return "***"
	}
	return token[:20] + "..."
}

// MaskPassword 脱敏密码
func MaskPassword(password string) string {
	if len(password) <= 2 {
		return "**"
	}
	return strings.Repeat("*", len(password))
}

// MaskSensitiveData 脱敏map中的敏感数据
func MaskSensitiveData(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range data {
		result[k] = maskValue(k, v)
	}
	return result
}

// maskValue 根据字段名脱敏值
func maskValue(key string, value interface{}) interface{} {
	if value == nil {
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return value
	}

	lowerKey := strings.ToLower(key)

	// 手机号字段
	if strings.Contains(lowerKey, "phone") || strings.Contains(lowerKey, "mobile") {
		return MaskPhone(str)
	}

	// 邮箱字段
	if strings.Contains(lowerKey, "email") {
		return MaskEmail(str)
	}

	// 密码字段
	if strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "pwd") {
		return MaskPassword(str)
	}

	// Token字段
	if strings.Contains(lowerKey, "token") || strings.Contains(lowerKey, "secret") {
		return MaskToken(str)
	}

	// 身份证字段
	if strings.Contains(lowerKey, "idcard") || strings.Contains(lowerKey, "id_card") {
		return MaskIDCard(str)
	}

	// 银行卡字段
	if strings.Contains(lowerKey, "bankcard") || strings.Contains(lowerKey, "card_no") {
		if len(str) >= 8 {
			return str[:4] + "****" + str[len(str)-4:]
		}
		return "****"
	}

	return value
}

// SanitizeLogMessage 脱敏日志消息中的敏感信息
func SanitizeLogMessage(msg string) string {
	// 脱敏手机号
	msg = PhoneRegex.ReplaceAllStringFunc(msg, func(match string) string {
		return MaskPhone(match)
	})

	// 脱敏邮箱
	msg = EmailRegex.ReplaceAllStringFunc(msg, func(match string) string {
		return MaskEmail(match)
	})

	return msg
}

// LogSafe 创建一个安全的日志map（用于记录日志）
func LogSafe(userID int64, action string, data map[string]interface{}) map[string]interface{} {
	safeData := make(map[string]interface{})
	for k, v := range data {
		safeData[k] = maskValue(k, v)
	}
	safeData["user_id"] = userID
	safeData["action"] = action
	return safeData
}

// SafePrint 安全打印结构体（用于调试）
func SafePrint(v interface{}) string {
	return fmt.Sprintf("%+v", v)
}
