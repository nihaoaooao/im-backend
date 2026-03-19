# CI/CD 配置说明

> 本文档说明 CI 流水线的配置和自定义方法

---

## 流水线概览

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   lint      │    │   test      │    │  security   │    │   build     │
│ 代码质量检查 │ -> │  单元测试   │ -> │  安全扫描   │ -> │  构建检查   │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
```

## 技术栈

- **后端**: Go 1.21 + Gin
- **数据库**: PostgreSQL 15
- **缓存**: Redis 7

## 触发条件

- **PR 到 main/develop**: 触发完整 CI
- **Push 到 main/develop**: 触发完整 CI

## 项目结构假设

```
项目根目录
├── backend/              # Go + Gin 后端
│   ├── go.mod
│   ├── main.go
│   └── ...
├── mobile/               # 移动端
│   ├── android/
│   │   └── build.gradle
│   └── ios/
│       └── *.xcodeproj
└── frontend/             # 前端
    └── package.json
```

---

## CI 流水线说明

### 1. Lint (代码质量检查)
- **Go**: golangci-lint / go vet / gofmt
- **移动端**: Gradle lint / Xcode build
- **前端**: ESLint

### 2. Test (测试)
- **单元测试**: `go test ./...`
- **集成测试**: 带数据库和缓存的测试
- **前端测试**: Jest / React Testing Library

### 3. Security (安全扫描)
- Go 漏洞扫描: golang/vuln-action
- Go 安全检查: gosec

### 4. Build (构建)
- Go 编译: `go build -o bin/server .`
- Android 构建: `./gradlew assembleDebug`
- 前端构建: `npm run build`

---

## 环境变量

CI 会自动配置以下环境变量：

```go
// 后端测试可用
DATABASE_URL=postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable
REDIS_URL=redis://localhost:6379
```

如需添加更多环境变量，在 workflow 中配置：

```yaml
- name: Run tests
  env:
    DATABASE_URL: postgres://user:pass@host:5432/db
    REDIS_URL: redis://host:6379
    API_KEY: ${{ secrets.API_KEY }}
```

---

## 测试标签

Go 项目支持集成测试标签：

```go
// +build integration

package integration

func TestIntegration(t *testing.T) {
    // 集成测试代码
}
```

运行集成测试：
```bash
go test -v -tags=integration ./...
```

---

## 自定义配置

### 1. 修改 Go 版本

```yaml
env:
  GO_VERSION: '1.22'
```

### 2. 修改数据库配置

```yaml
services:
  postgres:
    image: postgres:16
    env:
      POSTGRES_USER: myuser
      POSTGRES_PASSWORD: mypass
      POSTGRES_DB: mydb
```

### 3. 添加 Secret

1. 进入 GitHub 仓库 → Settings → Secrets and variables → Actions
2. 添加新的 Secret
3. 在 workflow 中使用：`${{ secrets.SECRET_NAME }}`

### 4. 添加额外的检查

```yaml
- name: Custom Check
  run: |
    cd backend
    # 你的自定义检查命令
```

---

## 常见问题

### Q: 如何跳过 CI？

```bash
git commit -m "feat: add feature [skip ci]"
```

### Q: CI 失败怎么办？

1. 点击 GitHub Actions 中失败的 job 查看日志
2. 本地修复问题
3. 推送更新

### Q: 单元测试和集成测试的区别？

- **单元测试**: 不依赖外部服务（数据库、Redis）
- **集成测试**: 需要 PostgreSQL 和 Redis 服务

### Q: 如何只运行后端 CI？

在 `.github/workflows/ci.yml` 中删除不需要的 job，或者使用路径过滤。

---

## 部署配置

如需添加自动部署：

```yaml
deploy:
  name: 部署
  runs-on: ubuntu-latest
  needs: [build]
  if: github.ref == 'refs/heads/main'
  steps:
    - name: Download artifact
      uses: actions/download-artifact@v3
      with:
        name: backend-binary

    - name: Deploy to server
      env:
        SSH_KEY: ${{ secrets.SSH_KEY }}
        SERVER_HOST: ${{ secrets.SERVER_HOST }}
      run: |
        # 部署脚本
        ./deploy.sh
```

---

## 下一步

1. **确认项目结构** - 确认 `backend/` 目录存在 Go 代码
2. **添加 Secrets** - 添加部署所需的敏感信息
3. **测试 CI** - 推送代码验证流水线
