package service

import (
	"encoding/json"
)

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(data)
}
