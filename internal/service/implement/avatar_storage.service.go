package implement

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	_interface "TaskFlow-Go/internal/service/interface"
	"TaskFlow-Go/internal/shared/apperror"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/disintegration/imaging"
)

type avatarStorageService struct {
	useS3  bool
	s3     *s3.Client
	bucket string
	localDir string
	baseURL  string
}

func NewAvatarStorageService() *avatarStorageService {
	s3Bucket := os.Getenv("S3_AVATAR_BUCKET")
	s3Region := os.Getenv("S3_REGION")
	localDir := os.Getenv("AVATAR_LOCAL_DIR")
	if localDir == "" {
		localDir = "uploads/avatars"
	}

	if s3Bucket != "" && s3Region != "" {
		cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(s3Region))
		if err == nil {
			return &avatarStorageService{
				useS3:  true,
				s3:     s3.NewFromConfig(cfg),
				bucket: s3Bucket,
			}
		}
	}

	baseURL := os.Getenv("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &avatarStorageService{
		useS3:    false,
		localDir: localDir,
		baseURL:  baseURL,
	}
}

func (s *avatarStorageService) Upload(userID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true}
	if !allowedExts[ext] {
		return "", apperror.ErrInvalidFileType
	}

	const maxSize int64 = 5 * 1024 * 1024
	if header.Size > maxSize {
		return "", apperror.ErrFileTooLarge
	}

	src, err := imaging.Decode(file)
	if err != nil {
		return "", apperror.ErrInvalidFileType
	}

	dst := imaging.Fill(src, 256, 256, imaging.Center, imaging.Lanczos)

	var buf bytes.Buffer
	if err := png.Encode(&buf, dst); err != nil {
		return "", apperror.ErrUploadFailed
	}

	key := fmt.Sprintf("avatars/%s.png", userID)

	if s.useS3 {
		_, err := s.s3.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(key),
			Body:        bytes.NewReader(buf.Bytes()),
			ContentType: aws.String("image/png"),
		})
		if err != nil {
			return "", apperror.ErrUploadFailed
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, os.Getenv("S3_REGION"), key), nil
	}

	if err := os.MkdirAll(s.localDir, 0755); err != nil {
		return "", apperror.ErrUploadFailed
	}
	filePath := filepath.Join(s.localDir, key)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", apperror.ErrUploadFailed
	}
	out, err := os.Create(filePath)
	if err != nil {
		return "", apperror.ErrUploadFailed
	}
	defer out.Close()

	if _, err := out.Write(buf.Bytes()); err != nil {
		return "", apperror.ErrUploadFailed
	}

	return fmt.Sprintf("%s/%s", s.baseURL, filepath.ToSlash(filePath)), nil
}

func (s *avatarStorageService) UploadBytes(userID string, data io.Reader, contentType string) (string, error) {
	key := fmt.Sprintf("avatars/%s.png", userID)

	if s.useS3 {
		_, err := s.s3.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket:      aws.String(s.bucket),
			Key:         aws.String(key),
			Body:        data,
			ContentType: aws.String(contentType),
		})
		if err != nil {
			return "", apperror.ErrUploadFailed
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucket, os.Getenv("S3_REGION"), key), nil
	}

	// Ensure the image is decoded for local save
	img, _, err := image.Decode(data)
	if err != nil {
		return "", apperror.ErrUploadFailed
	}

	if err := os.MkdirAll(s.localDir, 0755); err != nil {
		return "", apperror.ErrUploadFailed
	}
	filePath := filepath.Join(s.localDir, key)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return "", apperror.ErrUploadFailed
	}
	out, err := os.Create(filePath)
	if err != nil {
		return "", apperror.ErrUploadFailed
	}
	defer out.Close()

	if err := png.Encode(out, img); err != nil {
		return "", apperror.ErrUploadFailed
	}

	return fmt.Sprintf("%s/%s", s.baseURL, filepath.ToSlash(filePath)), nil
}

func (s *avatarStorageService) Delete(url string) error {
	if s.useS3 {
		return fmt.Errorf("s3 delete not implemented")
	}
	if url != "" {
		os.Remove("." + url)
	}
	return nil
}

var _ _interface.AvatarStorageService = (*avatarStorageService)(nil)
var _ _interface.ImageUploader = (*avatarStorageService)(nil)
