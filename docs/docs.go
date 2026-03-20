package docs

// Swagger 文档初始化文件
// 运行 swag init -g main.go -o docs 生成实际文档

import (
	"github.com/swaggo/swag"
)

// @title IM Backend API
// @version 1.0
// @description 高并发即时通讯系统后端 API 文档
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func init() {
	swag.Register("swagger", &swag.Info{
		Title:       "IM Backend API",
		Version:     "1.0",
		Description: "高并发即时通讯系统后端 API 文档",
		Host:        "localhost:8080",
		BasePath:    "/",
	})
}
