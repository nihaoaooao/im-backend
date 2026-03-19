package utils

import (
	"regexp"
	"strings"
)

// MaskSensitiveData 脱敏敏感数据
// 用于日志记录和API响应
func MaskSensitiveData(data string) string {
	if data == "" {
		return ""
	}

	// 电话号码脱敏: 13812345678 -> 138****5678
	phoneRegex := regexp.MustCompile(`(\d{3})\d{4}(\d{4})`)
	data = phoneRegex.ReplaceAllString(data, "$1****$2")

	// 邮箱脱敏: test@example.com -> t**t@e******.com
	emailRegex := regexp.MustCompile(`([a-zA-Z0-9._%+-]{1,2})[a-zA-Z0-9._%+-]*(@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
	data = emailRegex.ReplaceAllString(data, "$1**$2")

	// JWT Token 脱敏
	tokenRegex := regexp.MustCompile(`(eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*)`)
	data = tokenRegex.ReplaceAllString(data, "***")

	// 身份证号脱敏
	idCardRegex := regexp.MustCompile(`(\d{6})\d{8}(\d{4})`)
	data = idCardRegex.ReplaceAllString(data, "$1********$2")

	// 银行卡号脱敏
	bankCardRegex := regexp.MustCompile(`(\d{4})\d+(\d{4})`)
	data = bankCardRegex.ReplaceAllString(data, "$1****$2")

	return data
}

// MaskPhone 脱敏电话号码
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// MaskEmail 脱敏邮箱
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	username := parts[0]
	if len(username) <= 2 {
		return string(username[0]) + "***@" + parts[1]
	}
	return username[:1] + "***" + username[len(username)-1:] + "@" + parts[1]
}
