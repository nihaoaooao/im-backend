# 移动端测试部署指南

## 🎯 测试环境部署选项

### 选项 1：禁用证书锁定（推荐用于测试）

**iOS** - 修改 `APIConfig.swift`：
```swift
static let isCertificatePinningEnabled = false  // 测试环境禁用
```

**Android** - 修改 `build.gradle`：
```gradle
buildConfigField "boolean", "CERTIFICATE_PINNING_ENABLED", "false"
```

### 选项 2：使用 IP 地址自签名证书（如果有）

如果服务器使用自签名证书，需要获取该证书的指纹：

```bash
# 在服务器上运行
openssl x509 -in /path/to/server.crt -pubkey -noout | \
  openssl pkey -pubin -outform der | \
  openssl dgst -sha256 -binary | \
  openssl enc -base64
```

### 选项 3：配置域名和 HTTPS（生产环境推荐）

1. 为服务器配置域名（如 `api.im-test.com`）
2. 申请 Let's Encrypt 免费证书
3. 获取证书指纹并更新配置

---

## 📋 当前服务器状态

- **服务器 IP**: 129.226.74.230
- **HTTP 端口**: 8080
- **WebSocket 端口**: 8081
- **HTTPS**: 未配置（使用 IP 地址）

---

## ✅ 测试部署建议

对于当前测试环境，建议：

1. **暂时禁用证书锁定** - 因为使用 IP 地址没有标准 SSL 证书
2. **使用 HTTP 而非 HTTPS** - 测试环境可以接受
3. **生产环境再启用证书锁定** - 配置域名和正式证书后

---

## 🔒 生产环境准备清单

部署到生产前必须完成：

- [ ] 配置域名（如 `api.yourdomain.com`）
- [ ] 申请 SSL 证书（Let's Encrypt 或商业证书）
- [ ] 获取证书 SHA256 指纹
- [ ] 更新 iOS 和 Android 的指纹配置
- [ ] 启用证书锁定
- [ ] 全面测试验证
