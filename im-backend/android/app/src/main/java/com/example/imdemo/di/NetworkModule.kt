package com.example.imdemo.di

import android.content.Context
import com.example.imdemo.BuildConfig
import com.example.imdemo.data.remote.ApiService
import com.example.imdemo.util.CertificatePinningInterceptor
import dagger.Module
import dagger.Provides
import dagger.hilt.InstallIn
import dagger.hilt.android.qualifiers.ApplicationContext
import dagger.hilt.components.SingletonComponent
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit
import javax.inject.Singleton

/**
 * 网络模块 - 支持配置外部化
 * 
 * 配置来源优先级：
 * 1. BuildConfig（编译时配置）
 * 2. local.properties（本地配置，不提交到版本控制）
 */
@Module
@InstallIn(SingletonComponent::class)
object NetworkModule {

    /**
     * 提供 API 基础 URL
     * 从 BuildConfig 读取，支持不同环境的配置
     */
    @Provides
    fun provideBaseUrl(): String = BuildConfig.API_BASE_URL

    /**
     * 提供证书指纹列表
     * ⚠️ 安全警告：必须在生产环境部署前替换为真实的服务器证书指纹！
     * 
     * 获取真实指纹的方法：
     * ```bash
     * echo | openssl s_client -servername 129.226.74.230 -connect 129.226.74.230:443 2>/dev/null | \
     *   openssl x509 -pubkey -noout | \
     *   openssl pkey -pubin -outform der | \
     *   openssl dgst -sha256 -binary | \
     *   openssl enc -base64
     * ```
     */
    @Provides
    fun providePinnedCertificates(): List<String> = BuildConfig.PINNED_CERTIFICATES.toList()

    /**
     * 提供服务器主机名（用于证书锁定）
     */
    @Provides
    fun provideServerHost(): String = BuildConfig.SERVER_HOST

    @Provides
    @Singleton
    fun provideOkHttpClient(
        pinnedCertificates: List<String>,
        serverHost: String
    ): OkHttpClient {
        val loggingInterceptor = HttpLoggingInterceptor().apply {
            level = if (BuildConfig.DEBUG) {
                HttpLoggingInterceptor.Level.BODY
            } else {
                HttpLoggingInterceptor.Level.NONE
            }
        }

        val builder = OkHttpClient.Builder()
            .addInterceptor(loggingInterceptor)
            .connectTimeout(BuildConfig.REQUEST_TIMEOUT, TimeUnit.SECONDS)
            .readTimeout(BuildConfig.REQUEST_TIMEOUT, TimeUnit.SECONDS)
            .writeTimeout(BuildConfig.REQUEST_TIMEOUT, TimeUnit.SECONDS)

        // 根据配置启用证书锁定
        if (BuildConfig.CERTIFICATE_PINNING_ENABLED) {
            // 验证是否使用占位符
            val hasPlaceholder = pinnedCertificates.any { 
                it == "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=" 
            }
            
            if (hasPlaceholder) {
                if (BuildConfig.DEBUG) {
                    // 调试环境：警告但允许继续
                    println("⚠️ 警告：正在使用占位符证书指纹")
                    println("   请在生产环境部署前替换为真实的服务器证书指纹")
                } else {
                    // 生产环境：抛出异常
                    throw IllegalStateException(
                        "❌ 错误：生产环境不能使用占位符证书指纹。" +
                        "请在 build.gradle 中配置真实的服务器证书指纹"
                    )
                }
            }
            
            // 添加证书锁定拦截器
            builder.addInterceptor(CertificatePinningInterceptor(pinnedCertificates, serverHost))
        }

        return builder.build()
    }

    @Provides
    @Singleton
    fun provideRetrofit(okHttpClient: OkHttpClient, baseUrl: String): Retrofit {
        return Retrofit.Builder()
            .baseUrl(baseUrl)
            .client(okHttpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
    }

    @Provides
    @Singleton
    fun provideApiService(retrofit: Retrofit): ApiService {
        return retrofit.create(ApiService::class.java)
    }
}
