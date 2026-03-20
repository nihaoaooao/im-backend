import Foundation

/// API 配置
///
/// 支持从 Info.plist 读取配置，实现环境切换
struct APIConfig {
    
    // MARK: - 从 Info.plist 读取配置
    
    private static let infoDictionary: [String: Any] = {
        guard let dict = Bundle.main.infoDictionary else {
            fatalError("无法读取 Info.plist")
        }
        return dict
    }()
    
    // MARK: - 服务器配置
    
    /// API 基础 URL
    /// 从 Info.plist 的 API_BASE_URL 读取，默认为开发环境
    static let baseURL: String = {
        guard let url = infoDictionary["API_BASE_URL"] as? String else {
            // 默认使用开发环境
            return "https://129.226.74.230/api/v1"
        }
        return url
    }()
    
    /// WebSocket URL
    static let wsURL: String = {
        guard let url = infoDictionary["WS_URL"] as? String else {
            return "wss://129.226.74.230/ws"
        }
        return url
    }()
    
    /// 服务器主机名（用于证书锁定）
    static let serverHost: String = {
        guard let host = infoDictionary["SERVER_HOST"] as? String else {
            return "129.226.74.230"
        }
        return host
    }()
    
    // MARK: - 证书锁定配置
    
    /// 预置的证书 SHA256 指纹（十六进制格式，冒号分隔）
    ///
    /// ⚠️ 安全警告：部署前必须替换为真实的服务器证书指纹！
    ///
    /// 获取真实指纹的方法：
    /// ```bash
    /// # 方法1：使用 OpenSSL（推荐）
    /// echo | openssl s_client -servername 129.226.74.230 -connect 129.226.74.230:443 2>/dev/null | \
    ///   openssl x509 -pubkey -noout | \
    ///   openssl pkey -pubin -outform der | \
    ///   openssl dgst -sha256 -binary | \
    ///   openssl enc -base64
    ///
    /// # 方法2：使用浏览器
    /// # 1. 访问 https://129.226.74.230
    /// # 2. 点击地址栏的锁图标
    /// # 3. 查看证书详情，复制 SHA-256 指纹
    /// ```
    static let pinnedCertificateHashes: [String] = {
        // 从 Info.plist 读取，如果没有则使用默认值
        if let hashes = infoDictionary["PINNED_CERTIFICATE_HASHES"] as? [String] {
            return hashes
        }
        
        // ⚠️ 默认占位符指纹 - 必须在生产环境中替换！
        return [
            "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99"
        ]
    }()
    
    // MARK: - 请求配置
    
    /// 请求超时时间（秒）
    static let requestTimeout: TimeInterval = {
        if let timeout = infoDictionary["REQUEST_TIMEOUT"] as? TimeInterval {
            return timeout
        }
        return 30
    }()
    
    /// 是否启用证书锁定
    static let isCertificatePinningEnabled: Bool = {
        if let enabled = infoDictionary["CERTIFICATE_PINNING_ENABLED"] as? Bool {
            return enabled
        }
        return true
    }()
    
    /// 是否启用调试日志
    static let isDebugLoggingEnabled: Bool = {
        #if DEBUG
        return true
        #else
        return false
        #endif
    }()
}

// MARK: - 环境配置

extension APIConfig {
    
    /// 当前环境类型
    enum Environment {
        case development
        case staging
        case production
    }
    
    /// 当前环境
    static var currentEnvironment: Environment {
        #if DEBUG
        return .development
        #elseif STAGING
        return .staging
        #else
        return .production
        #endif
    }
    
    /// 环境名称
    static var environmentName: String {
        switch currentEnvironment {
        case .development:
            return "Development"
        case .staging:
            return "Staging"
        case .production:
            return "Production"
        }
    }
}

// MARK: - 配置验证

extension APIConfig {
    
    /// 验证配置是否正确
    /// 在应用启动时调用
    static func validate() {
        // 检查是否使用占位符指纹
        let placeholderFingerprints = [
            "AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99"
        ]
        
        let hasPlaceholder = pinnedCertificateHashes.contains { hash in
            placeholderFingerprints.contains(hash)
        }
        
        if hasPlaceholder {
            #if DEBUG
            print("⚠️ 警告：正在使用占位符证书指纹")
            print("   请在生产环境部署前替换为真实的服务器证书指纹")
            #else
            fatalError("❌ 错误：生产环境不能使用占位符证书指纹")
            #endif
        }
        
        // 验证 URL 格式
        guard URL(string: baseURL) != nil else {
            fatalError("❌ 错误：API_BASE_URL 格式不正确: \(baseURL)")
        }
        
        print("✅ API 配置验证通过")
        print("   环境: \(environmentName)")
        print("   API URL: \(baseURL)")
        print("   WebSocket URL: \(wsURL)")
        print("   证书锁定: \(isCertificatePinningEnabled ? "启用" : "禁用")")
        print("   预置指纹数量: \(pinnedCertificateHashes.count)")
    }
}
