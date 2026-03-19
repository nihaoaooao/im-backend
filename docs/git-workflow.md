# Git 工作流

> 本项目采用主干开发（Trunk-Based Development）模式

---

## 开始工作

### 1. 同步最新代码
```bash
git fetch origin
```

### 2. 创建功能分支
```bash
git checkout -b feat/my-feature origin/main
# 或
git checkout -b fix/bug-description origin/main
```

---

## 开发流程

### 日常开发
```bash
# 保存修改
git add .
git commit -m "feat: 添加新功能"

# 推送到远程（可多次执行）
git push origin feat/my-feature
```

### 保持分支最新
```bash
# 定期从 main 拉取最新代码
git fetch origin
git rebase origin/main

# 如果有冲突，解决后继续
git add .
git rebase --continue

# 强制推送（安全的 force push）
git push --force-with-lease origin feat/my-feature
```

---

## PR 前准备

### 1. 整理提交历史
```bash
# 交互式变基，压缩多个小提交
git rebase -i origin/main
```

### 2. 确保测试通过
```bash
# 本地运行测试
npm test  # 或其他测试命令
```

### 3. 最终推送
```bash
git push --force-with-lease origin feat/my-feature
```

---

## 创建 PR

在 GitHub/GitLab 上创建 Pull Request，填写：

### PR 模板
```markdown
## 变更内容
- 描述你的变更

## 测试说明
- 如何测试这些变更

## 关联 Issue
- 关联的相关 Issue

## 截图（如适用）
- UI 变更的截图
```

---

## 合并后清理

### 1. 切换到 main
```bash
git checkout main
```

### 2. 拉取最新代码
```bash
git pull origin main
```

### 3. 删除已合并的分支
```bash
git branch -d feat/my-feature
git push origin --delete feat/my-feature
```

---

## 紧急修复流程

### Hotfix 分支
```bash
# 从 main 创建 hotfix 分支
git checkout -b hotfix/critical-fix origin/main

# 修复后推送到远程
git push origin hotfix/critical-fix

# 创建 PR 合并到 main
```

---

## 常用命令速查

| 操作 | 命令 |
|------|------|
| 查看状态 | `git status` |
| 查看历史 | `git log --oneline --graph` |
| 切换分支 | `git checkout <branch>` |
| 创建分支 | `git checkout -b <branch>` |
| 暂存修改 | `git add .` |
| 提交 | `git commit -m "type: description"` |
| 推送 | `git push origin <branch>` |
| 拉取更新 | `git pull origin <branch>` |
| 变基更新 | `git rebase origin/main` |
| 强制推送 | `git push --force-with-lease` |

---

## 注意事项

⚠️ **永远不要对 main 分支执行：**
- `git push --force`（会覆盖远程历史）
- `git reset --hard`（会丢失代码）
- 直接提交（必须通过 PR）

⚠️ **提交前确认：**
- 提交信息格式正确
- 代码已经测试
- 没有包含敏感信息
