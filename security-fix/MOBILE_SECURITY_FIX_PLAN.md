# 移动端安全修复方案

## 🚨 关键安全问题

### P0 - 严重：iOS 证书锁定虚假实现

**问题文件**: `ios/IMDemo/IMDemo/Security/CertificatePinningDelegate.swift`

**当前问题代码**:
```swift
func urlSession(_ session: URLSession, didReceive challenge: URLAuthenticationChallenge,
                completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
    // 简化实现 - 实际应该验证服务器证书
    if let serverTrust = challenge.protectionSpace.serverTrust {
        let credential = URLCredential(trust: serverTrust)
        completionHandler(.useCredential, credential)
    } else {
        completionHandler(.cancelAuthenticationChallenge, nil)
    }
}
```

**问题分析**:
- 注释说"简化实现"，实际放行所有证书
- 没有进行任何证书指纹校验
- 攻击者可以使用任意证书进行中间人攻击

**修复方案**:
```swift
func urlSession(_ session: URLSession, didReceive challenge: URLAuthenticationChallenge,
                completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
    
    guard let serverTrust = challenge.protectionSpace.serverTrust,
          let certificateChain = SecTrustCopyCertificateChain(serverTrust) as? [SecCertificate],
          let serverCertificate = certificateChain.first else {
        completionHandler(.cancelAuthenticationChallenge, nil)
        return
    }
    
    // 获取服务器证书数据
    guard let serverCertificateData = SecCertificateCopyData(serverCertificate) as Data? else {
        completionHandler(.cancelAuthenticationChallenge, nil)
        return
    }
    
    // 计算证书 SHA256 指纹
    let serverCertificateHash = sha256(data: serverCertificateData)
    
    // 与预置的证书指纹比对
    if pinnedCertificateHashes.contains(serverCertificateHash) {
        let credential = URLCredential(trust: serverTrust)
        completionHandler(.useCredential, credential)
    } else {
        // 证书不匹配，拒绝连接
        print("❌ 证书锁定失败：服务器证书与预置指纹不匹配")
        completionHandler(.cancelAuthenticationChallenge, nil)
    }
}

private func sha256(data: Data) -> String {
    var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
    data.withUnsafeBytes {
        _ = CC_SHA256($0.baseAddress, CC_LONG(data.count), &hash)
    }
    return hash.map { String(format: "%02x", $0) }.joined(separator: ":")
}
```

---

## 🔧 需要获取的信息

### 1. 服务器证书指纹

需要获取服务器 `129.226.74.230` 的真实证书 SHA256 指纹：

```bash
# 获取证书指纹（在 Linux/Mac 上运行）
echo | openssl s_client -servername 129.226.74.230 -connect 129.226.74.230:443 2>/dev/null | \
  openssl x509 -pubkey -noout | \
  openssl pkey -pubin -outform der | \
  openssl dgst -sha256 -binary | \
  openssl enc -base64
```

### 2. 当前占位符指纹（需要替换）

**iOS** (`APIConfig.swift`):
```swift
static let pinnedCertificateHashes = [
    "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99"
]
```

**Android** (`NetworkModule.kt`):
```kotlin
private val pinnedCertificates = listOf(
    "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)
```

---

## 📋 修复任务清单

### 任务 1: 修复 iOS 证书锁定（P0 - 严重）
- [ ] 实现真实的证书指纹校验逻辑
- [ ] 添加 SHA256 指纹计算函数
- [ ] 拒绝不匹配的证书

### 任务 2: 更新证书指纹（P1 - 重要）
- [ ] 获取服务器真实证书指纹
- [ ] 更新 iOS `APIConfig.swift` 中的 `pinnedCertificateHashes`
- [ ] 更新 Android `NetworkModule.kt` 中的 `pinnedCertificates`

### 任务 3: 外部化 API 配置（P1 - 重要）
- [ ] iOS: 使用 `Info.plist` 或 `xcconfig` 文件存储 API 域名
- [ ] Android: 使用 `buildConfigField` 或 `local.properties`

---

## 👥 分工建议

| 角色 | 任务 |
|------|------|
| **移动工程师** | 实施 iOS/Android 代码修复，测试验证 |
| **后端/DevOps** | 提供服务器证书指纹，协助部署验证 |
| **安全工程师** | 修复后重新测试，确认漏洞已修复 |

---

## ✅ 验收标准

1. **iOS 证书锁定**:
   - 使用错误证书连接时，连接被拒绝
   - 使用正确证书连接时，连接成功
   - 单元测试通过

2. **证书指纹**:
   - 所有占位符已替换为真实指纹
   - iOS 和 Android 使用相同的指纹

3. **配置外部化**:
   - API 域名不在代码中硬编码
   - 支持 Debug/Release 环境切换

---

## 📝 备注

- 修复完成后需要重新进行安全测试
- 建议在修复前先在测试环境验证
- 考虑添加证书过期提醒机制
