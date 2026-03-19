# CI/CD 配置说明

> 本文档说明 CI 流水线的配置和自定义方法

---

## 流水线概览

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   lint      │    │   test      │    │   build     │
│ 代码质量检查 │ -> │  单元测试   │ -> │   构建检查   │
└─────────────┘    └─────────────┘    └─────────────┘
```

## 触发条件

- **PR 到 main/develop**: 触发完整 CI
- **Push 到 main/develop**: 触发完整 CI

## 项目结构假设

CI 配置假设项目目录结构如下：

```
项目根目录
├── backend/           # 后端服务
│   ├── go.mod         # Go 项目
│   ├── package.json   # Node.js 项目
│   └── pom.xml        # Java 项目
├── mobile/            # 移动端
│   ├── android/       # Android (Gradle)
│   │   └── build.gradle
│   └── ios/           # iOS (Xcode)
│       └── *.xcodeproj
└── frontend/          # 前端
    └── package.json
```

### 如果你的项目结构不同

请修改 `.github/workflows/ci.yml` 中的路径：

```yaml
# 修改示例
- name: Backend Lint
  run: |
    cd your-backend-folder  # 修改这里
    npm run lint
```

## 支持的技术栈

### 后端
- ✅ Go (`go.mod`)
- ✅ Node.js (`package.json`)
- ✅ Java (`pom.xml`)

### 移动端
- ✅ Android (Gradle)
- ✅ iOS (Xcode)

### 前端
- ✅ React/Vue/Angular (Node.js)

## 自定义步骤

### 1. 添加新的检查步骤

在 `lint` job 中添加：

```yaml
- name: Custom Lint
  run: |
    cd your-folder
    your-lint-command
```

### 2. 修改 Node.js 版本

```yaml
env:
  NODE_VERSION: '20'  # 改为 20
```

### 3. 添加环境变量

```yaml
- name: Run with env vars
  env:
    API_URL: ${{ secrets.API_URL }}
  run: |
    npm test
```

### 4. 添加 secret 变量

在 GitHub 仓库设置中添加：

1. Settings → Secrets and variables → Actions
2. 添加你的 secret（如 API_KEY）
3. 在 workflow 中使用：`${{ secrets.API_KEY }}`

## 常见问题

### Q: 如何跳过 CI？

```bash
git commit -m "chore: update [skip ci]"
```

### Q: CI 失败怎么办？

1. 点击失败的 job 查看日志
2. 修复本地问题
3. 推送更新：`git push`

### Q: 如何添加更多的测试？

```yaml
- name: Integration Tests
  run: |
    cd backend
    npm run test:integration
```

## 部署配置

如需添加自动部署，在 `ci.yml` 中添加：

```yaml
deploy:
  name: 部署
  runs-on: ubuntu-latest
  needs: [build]
  if: github.ref == 'refs/heads/main'
  steps:
    - name: Deploy to server
      run: |
        # 部署脚本
        ./deploy.sh
```

---

## 下一步

1. **确认项目结构** - 如果目录结构不同，修改 CI 配置
2. **添加 secret** - 添加部署所需的敏感信息
3. **测试 CI** - 推送代码验证流水线是否正常工作
