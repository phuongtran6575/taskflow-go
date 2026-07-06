package _interface

import (
	"io"
	"mime/multipart"
)

type AvatarStorageService interface {
	Upload(userID string, file multipart.File, header *multipart.FileHeader) (string, error)
	Delete(url string) error
}

// For direct use when the image has been processed (resized, converted)
type ImageUploader interface {
	UploadBytes(userID string, data io.Reader, contentType string) (string, error)
}
