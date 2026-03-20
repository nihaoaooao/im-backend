package service

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

// TestGenerateToken 测试 Token 生成
func TestGenerateToken(t *testing.T) {
	userService := &UserService{
		jwtSecret: "test-secret-key",
		jwtExpire: 24,
	}

	userID := int64(12345)
	username := "testuser"

	token, err := userService.generateToken(userID, username)
	if err != nil {
		t.Fatalf("generateToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("token should not be empty")
	}

	t.Logf("Generated token: %s", token)
}

// TestParseToken 测试 Token 解析
func TestParseToken(t *testing.T) {
	userService := &UserService{
		jwtSecret: "test-secret-key",
		jwtExpire: 24,
	}

	userID := int64(12345)
	username := "testuser"

	// 生成 token
	token, err := userService.generateToken(userID, username)
	if err != nil {
		t.Fatalf("generateToken failed: %v", err)
	}

	// 解析 token
	claims, err := userService.parseToken(token)
	if err != nil {
		t.Fatalf("parseToken failed: %v", err)
	}

	// 验证 claims
	if claims["user_id"] == nil {
		t.Fatal("user_id should be in claims")
	}

	if claims["username"] != username {
		t.Errorf("expected username %s, got %s", username, claims["username"])
	}
}

// TestGenerateRefreshToken 测试刷新 Token 生成
func TestGenerateRefreshToken(t *testing.T) {
	userService := &UserService{
		jwtSecret: "test-secret-key",
		jwtExpire: 24,
	}

	userID := int64(12345)

	token, err := userService.generateRefreshToken(userID)
	if err != nil {
		t.Fatalf("generateRefreshToken failed: %v", err)
	}

	if token == "" {
		t.Fatal("refresh token should not be empty")
	}

	// 解析验证
	claims, err := userService.parseToken(token)
	if err != nil {
		t.Fatalf("parseToken failed: %v", err)
	}

	if claims["type"] != "refresh" {
		t.Error("token type should be refresh")
	}
}

// TestInvalidToken 测试无效 Token
func TestInvalidToken(t *testing.T) {
	userService := &UserService{
		jwtSecret: "test-secret-key",
		jwtExpire: 24,
	}

	// 测试空 token
	_, err := userService.parseToken("")
	if err == nil {
		t.Error("should return error for empty token")
	}

	// 测试无效 token
	_, err = userService.parseToken("invalid-token")
	if err == nil {
		t.Error("should return error for invalid token")
	}
}

// TestPasswordHash 密码哈希测试
func TestPasswordHash(t *testing.T) {
	password := "testpassword123"

	// 加密
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	// 验证
	err = bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err != nil {
		t.Fatalf("password verification failed: %v", err)
	}

	// 验证错误密码
	err = bcrypt.CompareHashAndPassword(hash, []byte("wrongpassword"))
	if err == nil {
		t.Error("should return error for wrong password")
	}
}

// TestIsValidEmail 邮箱验证测试
func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"invalid", false},
		{"@example.com", false},
		{"test@", false},
		{"test@example", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isValidEmail(tt.email)
		if result != tt.valid {
			t.Errorf("isValidEmail(%s) = %v, expected %v", tt.email, result, tt.valid)
		}
	}
}

// TestIsValidPhone 手机号验证测试
func TestIsValidPhone(t *testing.T) {
	tests := []struct {
		phone string
		valid bool
	}{
		{"13800138000", true},
		{"+8613800138000", true},
		{"1234567890", false},
		{"abc", false},
		{"", false},
		{"123", false},
	}

	for _, tt := range tests {
		result := isValidPhone(tt.phone)
		if result != tt.valid {
			t.Errorf("isValidPhone(%s) = %v, expected %v", tt.phone, result, tt.valid)
		}
	}
}

// BenchmarkPasswordHash 密码哈希性能测试
func BenchmarkPasswordHash(b *testing.B) {
	password := "testpassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	}
}

// BenchmarkTokenGeneration Token 生成性能测试
func BenchmarkTokenGeneration(b *testing.B) {
	userService := &UserService{
		jwtSecret: "test-secret-key",
		jwtExpire: 24,
	}

	userID := int64(12345)
	username := "testuser"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userService.generateToken(userID, username)
	}
}

// BenchmarkTokenParsing Token 解析性能测试
func BenchmarkTokenParsing(b *testing.B) {
	userService := &UserService{
		jwtSecret: "test-secret-key",
		jwtExpire: 24,
	}

	userID := int64(12345)
	username := "testuser"
	token, _ := userService.generateToken(userID, username)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		userService.parseToken(token)
	}
}
