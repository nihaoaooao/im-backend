import Foundation

/// 认证服务
class AuthService {
    
    static let shared = AuthService()
    
    private let networkManager: NetworkManager
    
    init(networkManager: NetworkManager = .shared) {
        self.networkManager = networkManager
    }
    
    /// 用户注册
    func register(username: String, password: String, nickname: String?, completion: @escaping (Result<AuthResponse, NetworkError>) -> Void) {
        let request = RegisterRequest(username: username, password: password, nickname: nickname)
        networkManager.request(endpoint: "/auth/register", method: .post, body: request, completion: completion)
    }
    
    /// 用户登录
    func login(username: String, password: String, completion: @escaping (Result<AuthResponse, NetworkError>) -> Void) {
        let request = LoginRequest(username: username, password: password)
        networkManager.request(endpoint: "/auth/login", method: .post, body: request, completion: completion)
    }
    
    /// 刷新令牌
    func refreshToken(refreshToken: String, completion: @escaping (Result<TokenRefreshResponse, NetworkError>) -> Void) {
        let request = TokenRefreshRequest(refresh_token: refreshToken)
        networkManager.request(endpoint: "/auth/refresh", method: .post, body: request, completion: completion)
    }
}

// MARK: - 请求模型

struct RegisterRequest: Codable {
    let username: String
    let password: String
    let nickname: String?
}

struct LoginRequest: Codable {
    let username: String
    let password: String
}

struct TokenRefreshRequest: Codable {
    let refresh_token: String
}

// MARK: - 响应模型

struct AuthResponse: Codable {
    let user_id: Int
    let username: String
    let nickname: String?
    let access_token: String
    let refresh_token: String
    let expires_in: Int
}

struct TokenRefreshResponse: Codable {
    let access_token: String
    let expires_in: Int
}
