package com.example.imdemo.data.remote

import com.example.imdemo.data.model.*
import retrofit2.Response
import retrofit2.http.*

/**
 * API 服务接口
 */
interface ApiService {

    // ==================== 认证 ====================

    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): Response<AuthResponse>

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): Response<AuthResponse>

    @POST("auth/refresh")
    suspend fun refreshToken(@Body request: TokenRefreshRequest): Response<TokenRefreshResponse>

    // ==================== 用户 ====================

    @GET("users/profile")
    suspend fun getProfile(): Response<UserProfile>

    @PUT("users/profile")
    suspend fun updateProfile(@Body request: UpdateProfileRequest): Response<UserProfile>

    @GET("users/search")
    suspend fun searchUsers(@Query("keyword") keyword: String): Response<List<UserInfo>>

    // ==================== 好友 ====================

    @POST("friends/request")
    suspend fun sendFriendRequest(@Body request: FriendRequest): Response<Unit>

    @POST("friends/accept/{id}")
    suspend fun acceptFriendRequest(@Path("id") requestId: Int): Response<Unit>

    @GET("friends")
    suspend fun getFriends(): Response<List<FriendInfo>>

    // ==================== 消息 ====================

    @POST("messages")
    suspend fun sendMessage(@Body request: SendMessageRequest): Response<Message>

    @GET("messages/history")
    suspend fun getMessageHistory(
        @Query("user_id") userId: Int,
        @Query("page") page: Int = 1,
        @Query("page_size") pageSize: Int = 20
    ): Response<MessageHistoryResponse>

    @POST("messages/{id}/recall")
    suspend fun recallMessage(@Path("id") messageId: Int): Response<Unit>

    @POST("messages/read")
    suspend fun markMessagesRead(@Body request: MarkReadRequest): Response<Unit>

    // ==================== 群组 ====================

    @POST("groups")
    suspend fun createGroup(@Body request: CreateGroupRequest): Response<GroupInfo>

    @GET("groups")
    suspend fun getGroups(): Response<List<GroupInfo>>

    @POST("groups/{id}/members")
    suspend fun addGroupMember(
        @Path("id") groupId: Int,
        @Body request: AddMemberRequest
    ): Response<Unit>

    @GET("groups/{id}/messages")
    suspend fun getGroupMessages(
        @Path("id") groupId: Int,
        @Query("page") page: Int = 1,
        @Query("page_size") pageSize: Int = 20
    ): Response<MessageHistoryResponse>
}
