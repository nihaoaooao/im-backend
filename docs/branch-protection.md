# 分支保护规则配置

> 本文档记录分支保护规则的配置说明，供管理员在 GitHub/GitLab 后台配置

---

## main 分支保护规则

### GitHub 配置

```
Branch name pattern: main

✓ Require pull request reviews before merging
  - Required approving reviewers: 1

✓ Require status checks to pass before merging
  - Require branches to be up to date
  - Status checks: ci (CI流水线)

✓ Include administrators (禁用)

✓ Restrict who can push
  - 限制: 仅允许特定人员或机器人账号推送

✓ Allow force pushes (禁用)

✓ Allow deletions (禁用)
```

### GitLab 配置

```
Branch: main

✓ Protected

✓ Allowed to merge: Maintainers + Developers

✓ Allowed to push: Administrators only

✓ Require approval: Yes
  - Approvals required: 1

✓ Reset approvals on new push: Yes

✓ Require status checks: Yes
  - Require branches to be up to date before merging
```

---

## 分支命名规范

| 前缀 | 用途 | 示例 |
|------|------|------|
| `feat/` | 新功能 | `feat/user-auth` |
| `fix/` | Bug 修复 | `fix/login-error` |
| `docs/` | 文档更新 | `docs/api-spec` |
| `refactor/` | 代码重构 | `refactor/message-queue` |
| `perf/` | 性能优化 | `perf/optimization` |
| `test/` | 测试相关 | `test/integration` |
| `chore/` | 杂项/维护 | `chore/add-docker` |
| `release/` | 发布分支 | `release/v1.0.0` |
| `hotfix/` | 紧急修复 | `hotfix/critical-bug` |

---

## 合并策略

### 推荐：Squash Merge（压缩合并）

**优点：**
- 保持 main 分支历史线性干净
- 每个 PR 只产生一个 commit
- 便于追踪变更历史

**配置：**
- GitHub: Settings → Pull Requests → Allow squash merging
- GitLab: Settings → Merge Requests → Squashed commit

### 禁止

- ❌ 直接推送 main
- ❌ Merge commit（除非必要）
- ❌ Force push to main
