import Foundation

// MARK: - API 配置
// 安全修复版本 - 支持外部化配置
// 修复日期: 2026-03-20

enum APIEnvironment {
    case development
    case staging
    case production
}

class APIConfig {
    
    // MARK: - 单例
    static let shared = APIConfig()
    
    // MARK: - 当前环境
    private(set) var currentEnvironment: APIEnvironment = .production
    
    // MARK: - 配置属性
    var baseURL: String {
        return getBaseURL(for: currentEnvironment)
    }
    
    var wsURL: String {
        return getWebSocketURL(for: currentEnvironment)
    }
    
    // MARK: - 初始化
    private init() {
        // 从 Info.plist 读取环境配置
        loadConfiguration()
    }
    
    // MARK: - 配置加载
    
    private func loadConfiguration() {
        // 方式1: 从 Info.plist 读取 (推荐)
        if let envString = Bundle.main.object(forInfoDictionaryKey: "APIEnvironment") as? String {
            currentEnvironment = APIEnvironment.from(string: envString)
            print("[APIConfig] Loaded environment from Info.plist: \(envString)")
        }
        
        // 方式2: 从 UserDefaults 读取 (用于调试)
        #if DEBUG
        if let debugEnv = UserDefaults.standard.string(forKey: "debug_api_environment") {
            currentEnvironment = APIEnvironment.from(string: debugEnv)
            print("[APIConfig] DEBUG: Overridden environment: \(debugEnv)")
        }
        #endif
    }
    
    // MARK: - URL 配置
    
    private func getBaseURL(for environment: APIEnvironment) -> String {
        switch environment {
        case .development:
            // 开发环境 - 可从 Info.plist 或本地配置读取
            return Bundle.main.object(forInfoDictionaryKey: "APIBaseURL_Dev") as? String 
                ?? "http://localhost:8080"
            
        case .staging:
            // 测试环境
            return Bundle.main.object(forInfoDictionaryKey: "APIBaseURL_Staging") as? String
                ?? "https://staging.im.yourdomain.com"
            
        case .production:
            // 生产环境
            return Bundle.main.object(forInfoDictionaryKey: "APIBaseURL_Prod") as? String
                ?? "https://129.226.74.230:8080"
        }
    }
    
    private func getWebSocketURL(for environment: APIEnvironment) -> String {
        switch environment {
        case .development:
            return Bundle.main.object(forInfoDictionaryKey: "WSURL_Dev") as? String
                ?? "ws://localhost:8081"
            
        case .staging:
            return Bundle.main.object(forInfoDictionaryKey: "WSURL_Staging") as? String
                ?? "wss://staging.im.yourdomain.com/ws"
            
        case .production:
            return Bundle.main.object(forInfoDictionaryKey: "WSURL_Prod") as? String
                ?? "wss://129.226.74.230:8081"
        }
    }
    
    // MARK: - 公共方法
    
    /// 切换环境 (仅 DEBUG 模式可用)
    #if DEBUG
    func switchEnvironment(_ environment: APIEnvironment) {
        currentEnvironment = environment
        UserDefaults.standard.set(environment.stringValue, forKey: "debug_api_environment")
        print("[APIConfig] Switched to environment: \(environment)")
    }
    #endif
    
    /// 获取完整 API URL
    func apiURL(_ path: String) -> String {
        let cleanPath = path.hasPrefix("/") ? path : "/\(path)"
        return "\(baseURL)\(cleanPath)"
    }
    
    /// 获取 WebSocket URL (带 token)
    func webSocketURL(token: String) -> String {
        return "\(wsURL)?token=\(token)"
    }
}

// MARK: - APIEnvironment 扩展

extension APIEnvironment {
    static func from(string: String) -> APIEnvironment {
        switch string.lowercased() {
        case "dev", "development", "debug":
            return .development
        case "staging", "test":
            return .staging
        case "prod", "production", "release":
            return .production
        default:
            return .production
        }
    }
    
    var stringValue: String {
        switch self {
        case .development:
            return "development"
        case .staging:
            return "staging"
        case .production:
            return "production"
        }
    }
}

// MARK: - Info.plist 配置示例
/*
需要在 Info.plist 中添加以下配置:

<key>APIEnvironment</key>
<string>$(API_ENVIRONMENT)</string>

<key>APIBaseURL_Dev</key>
<string>http://localhost:8080</string>

<key>APIBaseURL_Staging</key>
<string>https://staging.im.yourdomain.com</string>

<key>APIBaseURL_Prod</key>
<string>https://129.226.74.230:8080</string>

<key>WSURL_Dev</key>
<string>ws://localhost:8081</string>

<key>WSURL_Staging</key>
<string>wss://staging.im.yourdomain.com/ws</string>

<key>WSURL_Prod</key>
<string>wss://129.226.74.230:8081</string>

Build Settings 中配置:
- Debug: API_ENVIRONMENT = development
- Release: API_ENVIRONMENT = production
*/
