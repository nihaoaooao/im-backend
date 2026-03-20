package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// AESEncrypt AES加密工具
// 使用AES-GCM模式进行加密，更安全

var (
	ErrKeyLengthInvalid = errors.New("key length must be 16, 24 or 32 bytes")
	ErrCiphertextShort  = errors.New("ciphertext too short")
)

// GenerateAESKey 生成随机AES密钥
func GenerateAESKey(length int) ([]byte, error) {
	if length != 16 && length != 24 && length != 32 {
		return nil, ErrKeyLengthInvalid
	}
	key := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// Encrypt 使用AES-GCM加密
// key: 16, 24或32字节的密钥
// plaintext: 要加密的明文
// 返回: 密文(base64编码)
func Encrypt(key []byte, plaintext []byte) (string, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", ErrKeyLengthInvalid
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 使用AES-GCM解密
// key: 16, 24或32字节的密钥（必须与加密时使用的密钥相同）
// ciphertext: 加密的密文(base64编码)
// 返回: 解密后的明文
func Decrypt(key []byte, ciphertext string) (string, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return "", ErrKeyLengthInvalid
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrCiphertextShort
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// EncryptString 加密字符串的便捷方法
func EncryptString(key []byte, plaintext string) (string, error) {
	return Encrypt(key, []byte(plaintext))
}

// DecryptString 解密字符串的便捷方法
func DecryptString(key []byte, ciphertext string) (string, error) {
	return Decrypt(key, ciphertext)
}
