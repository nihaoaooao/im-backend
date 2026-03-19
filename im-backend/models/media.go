package models

import (
	"encoding/json"
	"time"
)

// MediaType 媒体类型
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypeVoice MediaType = "voice"
	MediaTypeVideo MediaType = "video"
)

// Media 多媒体文件
type Media struct {
	ID           int64           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       int64           `gorm:"index;not null" json:"user_id"`
	Type         MediaType       `gorm:"type:varchar(20);not null" json:"type"`
	OriginalURL  string          `gorm:"type:varchar(500);not null" json:"original_url"`
	ThumbnailURL string          `gorm:"type:varchar(500)" json:"thumbnail_url,omitempty"`
	FileSize     int64           `gorm:"not null" json:"file_size"`
	Width        int             `json:"width,omitempty"`
	Height       int             `json:"height,omitempty"`
	Duration     float64         `json:"duration,omitempty"`
	Format       string          `gorm:"type:varchar(20)" json:"format,omitempty"`
	Metadata     json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`
	StorageKey   string          `gorm:"type:varchar(500);uniqueIndex" json:"storage_key"`
	CreatedAt    time.Time       `gorm:"index" json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

// TableName 指定表名
func (Media) TableName() string {
	return "media"
}

// MediaMetadata 媒体元数据
type MediaMetadata struct {
	// 图片特有
	Exif *ExifInfo `json:"exif,omitempty"`

	// 视频特有
	VideoInfo *VideoInfo `json:"video_info,omitempty"`

	// 音频特有
	AudioInfo *AudioInfo `json:"audio_info,omitempty"`
}

// ExifInfo EXIF信息
type ExifInfo struct {
	DateTime   string `json:"date_time,omitempty"`   // 拍摄时间
	GPSLat     string `json:"gps_lat,omitempty"`    // 纬度
	GPSLon     string `json:"gps_lon,omitempty"`    // 经度
	Make       string `json:"make,omitempty"`       // 设备厂商
	Model      string `json:"model,omitempty"`      // 设备型号
	Software   string `json:"software,omitempty"`   // 软件
	Orientation int   `json:"orientation,omitempty"` // 旋转角度
}

// VideoInfo 视频信息
type VideoInfo struct {
	Codec     string `json:"codec,omitempty"`     // 编码格式
	Bitrate   int64  `json:"bitrate,omitempty"`   // 码率
	FrameRate int    `json:"frame_rate,omitempty"` // 帧率
}

// AudioInfo 音频信息
type AudioInfo struct {
	Codec    string `json:"codec,omitempty"`    // 编码格式
	Bitrate  int64  `json:"bitrate,omitempty"`  // 码率
	SampleRate int   `json:"sample_rate,omitempty"` // 采样率
	Channels int    `json:"channels,omitempty"` // 声道数
}

// UserMediaQuota 用户媒体配额
type UserMediaQuota struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	TotalSize  int64     `gorm:"default:0" json:"total_size"`
	FileCount  int64     `gorm:"default:0" json:"file_count"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName 指定表名
func (UserMediaQuota) TableName() string {
	return "user_media_quota"
}
