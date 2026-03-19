# 贡献指南

感谢你对本项目的贡献！请遵循以下规范来保持项目质量。

---

## 分支规范

### 主分支
- `main` - 主分支，始终保持可部署状态，禁止直接推送

### 功能分支
- `feat/*` - 新功能开发
- `fix/*` - Bug 修复
- `docs/*` - 文档更新
- `refactor/*` - 代码重构
- `perf/*` - 性能优化
- `test/*` - 测试相关
- `chore/*` - 杂项/维护

### 特殊分支
- `release/*` - 发布分支
- `hotfix/*` - 紧急修复

---

## 提交规范

### 约定式提交 (Conventional Commits)

```
type(scope): description
```

**支持的类型：**

| 类型 | 说明 |
|------|------|
| `feat` | 新功能 |
| `fix` | Bug 修复 |
| `docs` | 文档更新 |
| `refactor` | 代码重构（不改变功能） |
| `perf` | 性能优化 |
| `test` | 测试相关 |
| `chore` | 杂项（构建、依赖更新等） |
| `style` | 代码格式（不影响功能） |
| `ci` | CI/CD 配置变更 |
| `build` | 构建系统变更 |

**示例：**
```
feat(user-auth): 添加用户注册功能
fix: 修复登录页面跳转错误
docs: 更新 API 文档
refactor(message): 优化消息队列逻辑
perf(websocket): 提升 WebSocket 连接性能
```

---

## PR 流程

1. **创建分支**
   ```bash
   git fetch origin
   git checkout -b feat/my-feature origin/main
   ```

2. **开发并提交**
   ```bash
   git add .
   git commit -m "feat: 添加新功能"
   ```

3. **保持分支更新**
   ```bash
   git fetch origin
   git rebase origin/main
   ```

4. **推送并创建 PR**
   ```bash
   git push -u origin feat/my-feature
   ```

5. **等待审查**
   - 至少 1 人审批
   - CI 流水线通过

6. **合并到 main**
   - 使用 Squash Merge
   - 删除功能分支

---

## 代码审查标准

- ✅ 代码功能正确
- ✅ 遵循项目编码规范
- ✅ 有必要的测试
- ✅ 没有安全漏洞
- ✅ 变更可追溯

---

## 协作沟通

- 使用 GitHub Issues 跟踪任务
- PR 描述要清晰说明变更内容
- 关联相关的 Issue
- 及时回复审查意见
