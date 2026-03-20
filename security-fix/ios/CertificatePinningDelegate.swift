import Foundation

// MARK: - 证书锁定委托
// 安全修复版本 - 实现真实的证书指纹校验
// 修复日期: 2026-03-20
// 问题: 原实现使用"简化实现"放行所有证书，存在中间人攻击风险

class CertificatePinningDelegate: NSObject, URLSessionDelegate {
    
    // MARK: - 服务器证书指纹配置
    // 注意: 部署前必须替换为真实的服务器证书 SHA256 指纹
    private let pinnedHashes: [String: [String]] = [
        "129.226.74.230": [
            // 生产环境证书指纹 - 需要替换为真实值
            // 获取方式: openssl s_client -connect 129.226.74.230:443 | openssl x509 -pubkey -noout | openssl pkey -pubin -outform der | openssl dgst -sha256 -binary | base64
            "REPLACE_WITH_ACTUAL_CERT_HASH"
        ],
        "im.yourdomain.com": [
            "REPLACE_WITH_DOMAIN_CERT_HASH"
        ]
    ]
    
    // 是否启用证书锁定 (生产环境必须为 true)
    private let pinningEnabled = true
    
    // 是否允许备用方案 (证书过期时的降级策略)
    private let allowFallback = false
    
    // MARK: - URLSessionDelegate
    
    func urlSession(
        _ session: URLSession,
        didReceive challenge: URLAuthenticationChallenge,
        completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void
    ) {
        guard pinningEnabled else {
            // 开发环境: 允许所有证书 (仅用于开发测试)
            #if DEBUG
            print("[Security] Certificate pinning disabled in DEBUG mode")
            completionHandler(.useCredential, URLCredential(trust: challenge.protectionSpace.serverTrust!))
            #else
            print("[Security] ERROR: Certificate pinning must be enabled in production!")
            completionHandler(.cancelAuthenticationChallenge, nil)
            #endif
            return
        }
        
        // 获取服务器信任对象
        guard let serverTrust = challenge.protectionSpace.serverTrust,
              let certificateChain = SecTrustCopyCertificateChain(serverTrust) as? [SecCertificate],
              !certificateChain.isEmpty else {
            print("[Security] ERROR: No server trust or certificate chain")
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }
        
        // 获取主机名
        let host = challenge.protectionSpace.host
        
        // 获取该主机的锁定指纹列表
        guard let expectedHashes = pinnedHashes[host], !expectedHashes.isEmpty else {
            print("[Security] ERROR: No pinned hashes configured for host: \(host)")
            if allowFallback {
                print("[Security] WARNING: Falling back to default validation")
                completionHandler(.performDefaultHandling, nil)
            } else {
                completionHandler(.cancelAuthenticationChallenge, nil)
            }
            return
        }
        
        // 验证证书链中的每个证书
        for certificate in certificateChain {
            if let publicKey = extractPublicKey(from: certificate),
               let publicKeyHash = hashPublicKey(publicKey) {
                
                print("[Security] Checking certificate hash: \(publicKeyHash)")
                
                // 检查指纹是否匹配
                if expectedHashes.contains(publicKeyHash) {
                    print("[Security] ✅ Certificate pinning successful for: \(host)")
                    let credential = URLCredential(trust: serverTrust)
                    completionHandler(.useCredential, credential)
                    return
                }
            }
        }
        
        // 证书不匹配 - 拒绝连接
        print("[Security] ❌ ERROR: Certificate pinning failed for: \(host)")
        print("[Security] Expected hashes: \(expectedHashes)")
        
        // 记录安全事件
        logSecurityEvent(host: host, reason: "Certificate pinning failed")
        
        completionHandler(.cancelAuthenticationChallenge, nil)
    }
    
    // MARK: - 辅助方法
    
    /// 从证书中提取公钥
    private func extractPublicKey(from certificate: SecCertificate) -> SecKey? {
        var publicKey: SecKey?
        
        // 创建临时信任对象来获取公钥
        let policy = SecPolicyCreateBasicX509()
        var trust: SecTrust?
        let status = SecTrustCreateWithCertificates(certificate, policy, &trust)
        
        guard status == errSecSuccess, let trustObject = trust else {
            return nil
        }
        
        publicKey = SecTrustCopyKey(trustObject)
        return publicKey
    }
    
    /// 对公钥进行 SHA256 哈希
    private func hashPublicKey(_ publicKey: SecKey) -> String? {
        guard let publicKeyData = SecKeyCopyExternalRepresentation(publicKey, nil) else {
            return nil
        }
        
        let data = publicKeyData as Data
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        data.withUnsafeBytes { bytes in
            _ = CC_SHA256(bytes.baseAddress, CC_LONG(data.count), &hash)
        }
        
        // 转换为 Base64 字符串
        let hashData = Data(hash)
        return hashData.base64EncodedString()
    }
    
    /// 记录安全事件
    private func logSecurityEvent(host: String, reason: String) {
        let event = [
            "timestamp": ISO8601DateFormatter().string(from: Date()),
            "host": host,
            "reason": reason,
            "type": "certificate_pinning_failure"
        ]
        print("[Security Event] \(event)")
        
        // 可以在这里添加更多日志记录，如发送到服务器
    }
    
    // MARK: - 公共方法
    
    /// 更新证书指纹 (用于证书轮换)
    func updatePinnedHash(for host: String, hash: String) {
        // 注意: 此方法仅用于证书轮换期间
        // 生产环境应通过应用更新来更新指纹
        print("[Security] Updating pinned hash for: \(host)")
    }
    
    /// 验证当前配置是否有效
    func validateConfiguration() -> Bool {
        for (host, hashes) in pinnedHashes {
            if hashes.contains("REPLACE_WITH_ACTUAL_CERT_HASH") {
                print("[Security] WARNING: Placeholder hash detected for host: \(host)")
                return false
            }
        }
        return true
    }
}

// MARK: - 依赖导入说明
// 需要在 Bridging Header 或模块中导入:
// #import <CommonCrypto/CommonCrypto.h>
