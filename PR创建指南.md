# 后端工程师 - PR 创建指南

## 🚀 快速操作步骤

### 步骤 1：打开 GitHub 网页
访问仓库地址：
```
https://github.com/nihaoaooao/im-backend
```

### 步骤 2：创建 Pull Request

1. 点击页面上的 **"Pull requests"** 标签
2. 点击绿色的 **"New pull request"** 按钮
3. 选择分支：
   - **base** (左侧): 选择 `main`
   - **compare** (右侧): 选择 `fix/security-joint-issues`
4. 点击 **"Create pull request"**

### 步骤 3：填写 PR 信息

**标题：**
```
fix: 修复多媒体上传安全问题及编译错误
```

**描述：**
```markdown
## 修复内容

### 安全问题修复（7个）
- ✅ 修复 SQL 注入漏洞
- ✅ 修复 XSS 攻击漏洞
- ✅ 修复文件上传类型验证
- ✅ 修复文件大小限制
- ✅ 修复 Token 泄露问题
- ✅ 修复敏感信息日志打印
- ✅ 修复不安全的文件存储路径

### 审查问题修复（2个）
- ✅ 修复错误处理不完善
- ✅ 修复代码注释缺失

### 编译错误修复（7个）
- ✅ 修复类型不匹配错误
- ✅ 修复导入缺失错误
- ✅ 修复语法错误
- ✅ 修复依赖版本冲突

## 审查状态
- ✅ 代码安全最终审查已通过
- ✅ 自测通过

## 检查清单
- [x] 安全问题已修复
- [x] 审查问题已修复
- [x] 编译错误已修复
- [x] 代码已测试
```

### 步骤 4：创建 PR
点击 **"Create pull request"** 按钮

### 步骤 5：合并 PR
1. 等待 GitHub Actions 检查通过（如果有）
2. 点击 **"Merge pull request"** 按钮
3. 点击 **"Confirm merge"** 确认合并

---

## ⚠️ 注意事项

- **不要直接推送到 main 分支**，必须通过 PR 合并
- 如果提示 "protected branch update failed"，说明必须通过 PR
- 如果没有 Merge 按钮，请联系仓库管理员

## 🔗 直接访问链接

可以直接访问以下链接创建 PR：
```
https://github.com/nihaoaooao/im-backend/compare/main...fix/security-joint-issues
```

---

**当前分支**: `fix/security-joint-issues`  
**目标分支**: `main`  
**状态**: 代码已提交，等待创建 PR
