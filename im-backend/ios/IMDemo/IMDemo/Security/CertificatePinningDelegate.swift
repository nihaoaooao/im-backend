import Foundation
import CommonCrypto

/// 证书锁定委托 - 修复后的安全版本
/// 
/// 实现了完整的 SSL 证书锁定功能：
/// 1. 提取服务器证书
/// 2. 计算证书 SHA256 指纹
/// 3. 与预置的证书指纹进行比对
/// 4. 拒绝不匹配的证书
class CertificatePinningDelegate: NSObject, URLSessionDelegate {
    
    /// 预置的证书 SHA256 指纹（十六进制格式，冒号分隔）
    /// ⚠️ 重要：部署前必须替换为真实的服务器证书指纹
    ///
    /// 获取真实指纹的方法：
    /// ```bash
    /// echo | openssl s_client -servername 129.226.74.230 -connect 129.226.74.230:443 2>/dev/null | \
    ///   openssl x509 -pubkey -noout | \
    ///   openssl pkey -pubin -outform der | \
    ///   openssl dgst -sha256 -binary | \
    ///   openssl enc -base64
    /// ```
    private let pinnedCertificateHashes: [String]
    
    /// 是否严格模式（失败时拒绝连接）
    private let strictMode: Bool
    
    /// 初始化
    /// - Parameters:
    ///   - pinnedHashes: 预置的证书指纹列表
    ///   - strictMode: 是否启用严格模式（默认 true）
    init(pinnedHashes: [String] = APIConfig.pinnedCertificateHashes, strictMode: Bool = true) {
        self.pinnedCertificateHashes = pinnedHashes
        self.strictMode = strictMode
        super.init()
    }
    
    /// 处理服务器认证挑战
    /// 
    /// 这是证书锁定的核心逻辑：
    /// 1. 获取服务器证书链
    /// 2. 提取服务器证书
    /// 3. 计算证书的 SHA256 指纹
    /// 4. 与预置指纹比对
    /// 5. 匹配则允许连接，不匹配则拒绝
    func urlSession(_ session: URLSession, didReceive challenge: URLAuthenticationChallenge,
                    completionHandler: @escaping (URLSession.AuthChallengeDisposition, URLCredential?) -> Void) {
        
        // 获取服务器信任对象
        guard let serverTrust = challenge.protectionSpace.serverTrust else {
            print("❌ 证书锁定失败：无法获取服务器信任对象")
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }
        
        // 获取证书链（iOS 15+ 使用 SecTrustCopyCertificateChain）
        guard let certificateChain = SecTrustCopyCertificateChain(serverTrust) as? [SecCertificate],
              let serverCertificate = certificateChain.first else {
            print("❌ 证书锁定失败：无法获取服务器证书")
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }
        
        // 获取证书数据
        guard let serverCertificateData = SecCertificateCopyData(serverCertificate) as Data? else {
            print("❌ 证书锁定失败：无法读取证书数据")
            completionHandler(.cancelAuthenticationChallenge, nil)
            return
        }
        
        // 计算证书指纹
        let serverCertificateHash = sha256Fingerprint(data: serverCertificateData)
        
        // 验证指纹是否匹配
        if pinnedCertificateHashes.contains(serverCertificateHash) {
            // 证书匹配，允许连接
            print("✅ 证书锁定验证通过")
            let credential = URLCredential(trust: serverTrust)
            completionHandler(.useCredential, credential)
        } else {
            // 证书不匹配
            print("❌ 证书锁定失败：")
            print("   服务器证书指纹: \(serverCertificateHash)")
            print("   预置指纹列表: \(pinnedCertificateHashes)")
            
            if strictMode {
                // 严格模式：拒绝连接
                print("   操作：拒绝连接（严格模式）")
                completionHandler(.cancelAuthenticationChallenge, nil)
            } else {
                // 非严格模式：允许连接（仅用于调试）
                print("⚠️  警告：允许不安全的连接（非严格模式，仅用于调试）")
                let credential = URLCredential(trust: serverTrust)
                completionHandler(.useCredential, credential)
            }
        }
    }
    
    /// 计算数据的 SHA256 指纹（十六进制格式，冒号分隔）
    ///
    /// 例如: "AB:CD:EF:12:34:56:..."
    private func sha256Fingerprint(data: Data) -> String {
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        data.withUnsafeBytes { buffer in
            _ = CC_SHA256(buffer.baseAddress, CC_LONG(data.count), &hash)
        }
        return hash.map { String(format: "%02X", $0) }.joined(separator: ":")
    }
    
    /// 计算数据的 SHA256 哈希（Base64 格式）
    ///
    /// 用于与 OpenSSL 输出的格式对比
    func sha256Base64(data: Data) -> String {
        var hash = [UInt8](repeating: 0, count: Int(CC_SHA256_DIGEST_LENGTH))
        data.withUnsafeBytes { buffer in
            _ = CC_SHA256(buffer.baseAddress, CC_LONG(data.count), &hash)
        }
        return Data(hash).base64EncodedString()
    }
}

// MARK: - 证书锁定调试工具

extension CertificatePinningDelegate {
    
    /// 调试方法：打印服务器证书信息
    /// 用于获取服务器证书指纹
    func debugPrintCertificateInfo(forHost host: String) {
        print("=== 证书调试信息 ===")
        print("主机: \(host)")
        print("预置指纹数量: \(pinnedCertificateHashes.count)")
        print("严格模式: \(strictMode)")
        
        // 获取真实指纹的方法提示
        print("\n获取真实证书指纹的方法:")
        print("1. 使用浏览器访问: https://\(host)")
        print("2. 点击地址栏的锁图标")
        print("3. 查看证书详情")
        print("4. 复制 SHA-256 指纹")
        print("\n或使用命令行:")
        print("echo | openssl s_client -servername \(host) -connect \(host):443 2>/dev/null | \\")
        print("  openssl x509 -pubkey -noout | \\")
        print("  openssl pkey -pubin -outform der | \\")
        print("  openssl dgst -sha256 -binary | \\")
        print("  openssl enc -base64")
    }
}
