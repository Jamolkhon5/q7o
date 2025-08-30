package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type UploadConfig struct {
	MaxFileSize      int64    // Maximum file size in bytes
	AllowedMimeTypes []string // Allowed MIME types
	UploadPath       string   // Local upload path
	MaxWidth         int      // Maximum image width
	MaxHeight        int      // Maximum image height
	Quality          int      // JPEG quality (1-100)
}

func LoadUploadConfig() UploadConfig {
	maxSizeStr := getEnv("UPLOAD_MAX_SIZE", "5242880") // 5MB default
	maxSize, _ := strconv.ParseInt(maxSizeStr, 10, 64)

	maxWidthStr := getEnv("UPLOAD_MAX_WIDTH", "1024")
	maxWidth, _ := strconv.Atoi(maxWidthStr)

	maxHeightStr := getEnv("UPLOAD_MAX_HEIGHT", "1024")
	maxHeight, _ := strconv.Atoi(maxHeightStr)

	qualityStr := getEnv("UPLOAD_QUALITY", "85")
	quality, _ := strconv.Atoi(qualityStr)

	uploadPath := getEnv("UPLOAD_PATH", "./uploads")
	
	// Ensure upload directory exists
	if err := os.MkdirAll(filepath.Join(uploadPath, "avatars"), 0755); err != nil {
		// Log error but continue with default path
		uploadPath = "./uploads"
		os.MkdirAll(filepath.Join(uploadPath, "avatars"), 0755)
	}

	allowedTypes := strings.Split(getEnv("UPLOAD_ALLOWED_TYPES", "image/jpeg,image/png,image/gif,image/webp"), ",")

	return UploadConfig{
		MaxFileSize:      maxSize,
		AllowedMimeTypes: allowedTypes,
		UploadPath:       uploadPath,
		MaxWidth:         maxWidth,
		MaxHeight:        maxHeight,
		Quality:          quality,
	}
}

func (c UploadConfig) IsAllowedMimeType(mimeType string) bool {
	for _, allowed := range c.AllowedMimeTypes {
		if strings.TrimSpace(allowed) == mimeType {
			return true
		}
	}
	return false
}