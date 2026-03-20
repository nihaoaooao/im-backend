import Foundation

/// 令牌管理器
class TokenManager {
    
    static let shared = TokenManager()
    
    private let keychain = KeychainService()
    
    private let accessTokenKey = "access_token"
    private let refreshTokenKey = "refresh_token"
    private let userIdKey = "user_id"
    
    /// 当前访问令牌
    var accessToken: String? {
        get { keychain.get(accessTokenKey) }
        set { 
            if let token = newValue {
                keychain.set(token, forKey: accessTokenKey)
            } else {
                keychain.delete(accessTokenKey)
            }
        }
    }
    
    /// 刷新令牌
    var refreshToken: String? {
        get { keychain.get(refreshTokenKey) }
        set {
            if let token = newValue {
                keychain.set(token, forKey: refreshTokenKey)
            } else {
                keychain.delete(refreshTokenKey)
            }
        }
    }
    
    /// 当前用户ID
    var userId: Int? {
        get {
            guard let idString = keychain.get(userIdKey),
                  let id = Int(idString) else { return nil }
            return id
        }
        set {
            if let id = newValue {
                keychain.set(String(id), forKey: userIdKey)
            } else {
                keychain.delete(userIdKey)
            }
        }
    }
    
    /// 是否已登录
    var isLoggedIn: Bool {
        accessToken != nil
    }
    
    /// 保存登录信息
    func saveAuth(_ auth: AuthResponse) {
        accessToken = auth.access_token
        refreshToken = auth.refresh_token
        userId = auth.user_id
    }
    
    /// 清除登录信息
    func clearAuth() {
        accessToken = nil
        refreshToken = nil
        userId = nil
    }
}

// MARK: - Keychain 服务

class KeychainService {
    
    func set(_ value: String, forKey key: String) {
        guard let data = value.data(using: .utf8) else { return }
        
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecValueData as String: data
        ]
        
        // 删除旧值
        SecItemDelete(query as CFDictionary)
        
        // 添加新值
        SecItemAdd(query as CFDictionary, nil)
    }
    
    func get(_ key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne
        ]
        
        var result: AnyObject?
        SecItemCopyMatching(query as CFDictionary, &result)
        
        guard let data = result as? Data,
              let value = String(data: data, encoding: .utf8) else { return nil }
        
        return value
    }
    
    func delete(_ key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrAccount as String: key
        ]
        
        SecItemDelete(query as CFDictionary)
    }
}
