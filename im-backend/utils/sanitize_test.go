package utils

import (
	"testing"
)

// TestMaskSensitiveData 测试敏感数据脱敏
func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "phone number",
			input:    "13812345678",
			expected: "138****5678",
		},
		{
			name:     "email",
			input:    "test@example.com",
			expected: "t**t@e******.com",
		},
		{
			name:     "short email",
			input:    "a@b.com",
			expected: "a***@b.com",
		},
		{
			name:     "ID card",
			input:    "110101199001011234",
			expected: "110101********1234",
		},
		{
			name:     "bank card",
			input:    "6222021234567890123",
			expected: "6222*******0123",
		},
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "text with phone",
			input:    "My phone is 13912345678",
			expected: "My phone is 139****5678",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveData(tt.input)
			if result != tt.expected {
				t.Errorf("MaskSensitiveData(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestMaskPhone 测试电话脱敏
func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"13812345678", "138****5678"},
		{"12345678901", "123****8901"},
		{"12345", "12345"}, // 不足11位
	}

	for _, tt := range tests {
		result := MaskPhone(tt.input)
		if result != tt.expected {
			t.Errorf("MaskPhone(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestMaskEmail 测试邮箱脱敏
func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "t***t@e******.com"},
		{"a@b.com", "a***@b.com"},
		{"ab@c.com", "a**b@c.com"},
	}

	for _, tt := range tests {
		result := MaskEmail(tt.input)
		if result != tt.expected {
			t.Errorf("MaskEmail(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
