import Foundation

/// 网络管理器
class NetworkManager {
    
    static let shared = NetworkManager()
    
    private let session: URLSession
    private let pinningDelegate = CertificatePinningDelegate()
    
    init() {
        let configuration = URLSessionConfiguration.default
        configuration.timeoutIntervalForRequest = APIConfig.requestTimeout
        configuration.timeoutIntervalForResource = APIConfig.requestTimeout * 2
        
        // 使用证书锁定委托创建 session
        self.session = URLSession(configuration: configuration, delegate: pinningDelegate, delegateQueue: nil)
    }
    
    /// 发送网络请求
    func request<T: Decodable>(endpoint: String, method: HTTPMethod = .get, 
                               body: Encodable? = nil, completion: @escaping (Result<T, NetworkError>) -> Void) {
        
        guard let url = URL(string: APIConfig.baseURL + endpoint) else {
            completion(.failure(.invalidURL))
            return
        }
        
        var request = URLRequest(url: url)
        request.httpMethod = method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        
        // 添加认证头
        if let token = TokenManager.shared.accessToken {
            request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        
        // 添加请求体
        if let body = body {
            do {
                request.httpBody = try JSONEncoder().encode(body)
            } catch {
                completion(.failure(.encodingError))
                return
            }
        }
        
        // 发送请求
        let task = session.dataTask(with: request) { data, response, error in
            DispatchQueue.main.async {
                if let error = error {
                    completion(.failure(.networkError(error.localizedDescription)))
                    return
                }
                
                guard let httpResponse = response as? HTTPURLResponse else {
                    completion(.failure(.invalidResponse))
                    return
                }
                
                guard let data = data else {
                    completion(.failure(.noData))
                    return
                }
                
                // 检查状态码
                switch httpResponse.statusCode {
                case 200...299:
                    do {
                        let decoded = try JSONDecoder().decode(T.self, from: data)
                        completion(.success(decoded))
                    } catch {
                        completion(.failure(.decodingError))
                    }
                case 401:
                    completion(.failure(.unauthorized))
                default:
                    completion(.failure(.serverError(httpResponse.statusCode)))
                }
            }
        }
        
        task.resume()
    }
}

// MARK: - HTTP 方法

enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
    case patch = "PATCH"
}

// MARK: - 网络错误

enum NetworkError: Error {
    case invalidURL
    case encodingError
    case networkError(String)
    case invalidResponse
    case noData
    case decodingError
    case unauthorized
    case serverError(Int)
}
