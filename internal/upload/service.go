package upload

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"golang.org/x/image/webp"

	"q7o/config"
)

type Service struct {
	config config.UploadConfig
}

func NewService(cfg config.UploadConfig) *Service {
	return &Service{
		config: cfg,
	}
}

type UploadResult struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
}

func (s *Service) UploadAvatar(file *multipart.FileHeader) (*UploadResult, error) {
	// Check file size
	if file.Size > s.config.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds limit of %d bytes", s.config.MaxFileSize)
	}

	// Open file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	// Detect content type
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read file header: %w", err)
	}

	// Reset file pointer
	src.Seek(0, 0)

	mimeType := getMimeType(buffer)
	if !s.config.IsAllowedMimeType(mimeType) {
		return nil, fmt.Errorf("unsupported file type: %s", mimeType)
	}

	// Generate unique filename
	ext := getFileExtension(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", uuid.New().String(), time.Now().Unix(), ext)
	
	// Create full path
	avatarDir := filepath.Join(s.config.UploadPath, "avatars")
	fullPath := filepath.Join(avatarDir, filename)

	// Process image
	processedSize, err := s.processAndSaveImage(src, fullPath, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %w", err)
	}

	// Generate URL
	url := fmt.Sprintf("/uploads/avatars/%s", filename)

	return &UploadResult{
		Filename: filename,
		URL:      url,
		Size:     processedSize,
	}, nil
}

func (s *Service) DeleteAvatar(filename string) error {
	if filename == "" {
		return nil // Nothing to delete
	}

	// Extract filename from URL if needed
	if strings.Contains(filename, "/") {
		filename = filepath.Base(filename)
	}

	fullPath := filepath.Join(s.config.UploadPath, "avatars", filename)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return nil // File doesn't exist, nothing to delete
	}

	// Delete file
	return os.Remove(fullPath)
}

func (s *Service) processAndSaveImage(src io.Reader, destPath string, mimeType string) (int64, error) {
	// Decode image based on type
	var img image.Image
	var err error

	switch mimeType {
	case "image/jpeg":
		img, err = jpeg.Decode(src)
	case "image/png":
		img, err = png.Decode(src)
	case "image/gif":
		img, err = gif.Decode(src)
	case "image/webp":
		img, err = webp.Decode(src)
	default:
		return 0, fmt.Errorf("unsupported image format: %s", mimeType)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if necessary
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width > s.config.MaxWidth || height > s.config.MaxHeight {
		// Calculate new dimensions maintaining aspect ratio
		if width > height {
			height = int(float64(height) * (float64(s.config.MaxWidth) / float64(width)))
			width = s.config.MaxWidth
		} else {
			width = int(float64(width) * (float64(s.config.MaxHeight) / float64(height)))
			height = s.config.MaxHeight
		}
		
		img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
	}

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Encode and save
	switch mimeType {
	case "image/jpeg":
		err = jpeg.Encode(dst, img, &jpeg.Options{Quality: s.config.Quality})
	case "image/png":
		err = png.Encode(dst, img)
	case "image/gif":
		err = gif.Encode(dst, img, nil)
	case "image/webp":
		// Note: webp encoding requires additional library
		// For now, convert to JPEG
		err = jpeg.Encode(dst, img, &jpeg.Options{Quality: s.config.Quality})
	}

	if err != nil {
		os.Remove(destPath) // Clean up on error
		return 0, fmt.Errorf("failed to encode image: %w", err)
	}

	// Get file size
	info, err := dst.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}

	return info.Size(), nil
}

func getMimeType(buffer []byte) string {
	if len(buffer) < 12 {
		return ""
	}

	// JPEG
	if buffer[0] == 0xFF && buffer[1] == 0xD8 && buffer[2] == 0xFF {
		return "image/jpeg"
	}

	// PNG
	if buffer[0] == 0x89 && buffer[1] == 0x50 && buffer[2] == 0x4E && buffer[3] == 0x47 {
		return "image/png"
	}

	// GIF
	if buffer[0] == 0x47 && buffer[1] == 0x49 && buffer[2] == 0x46 {
		return "image/gif"
	}

	// WebP
	if len(buffer) >= 12 &&
		buffer[0] == 0x52 && buffer[1] == 0x49 && buffer[2] == 0x46 && buffer[3] == 0x46 &&
		buffer[8] == 0x57 && buffer[9] == 0x45 && buffer[10] == 0x42 && buffer[11] == 0x50 {
		return "image/webp"
	}

	return ""
}

func getFileExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ".jpg" // Default to jpg if no extension
	}
	return strings.ToLower(ext)
}