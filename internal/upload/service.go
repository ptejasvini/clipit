package upload

import (
	"bytes"
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	ChunkSize   int64 = 512 * 1024
	MaxFileSize int64 = 2 * 1024 * 1024
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrTooLarge     = errors.New("file too large")
)

type Service struct {
	store *Store
	mu    sync.Mutex
}

func NewService(store *Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) CreateUpload(name string, size int64) (Upload, error) {
	if size <= 0 {
		return Upload{}, ErrInvalidInput
	}
	if size > MaxFileSize {
		return Upload{}, ErrTooLarge
	}
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || len(name) > 200 {
		return Upload{}, ErrInvalidInput
	}

	totalParts := int((size + ChunkSize - 1) / ChunkSize)

	upload := Upload{
		ID:         uuid.NewString(),
		Name:       name,
		Size:       size,
		TotalParts: totalParts,
		ChunkSize:  ChunkSize,
		Parts:      make(map[int]bool),
		Status:     "uploading",
		CreatedAt:  time.Now().UTC(),
	}
	if err := os.Mkdir(s.store.UploadDirectory(upload.ID), 0o750); err != nil {
		return Upload{}, err
	}
	if err := s.store.Save(upload); err != nil {
		os.RemoveAll(s.store.UploadDirectory(upload.ID))
		return Upload{}, err
	}
	return upload, nil
}

func (s *Service) UploadPart(uploadID string, partNumber int, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.store.Get(uploadID)
	if !ok {
		return ErrNotFound
	}
	if upload.Status != "uploading" {
		return ErrConflict
	}
	if partNumber < 0 || partNumber >= upload.TotalParts {
		return ErrInvalidInput
	}

	expectedSize := upload.ChunkSize
	if partNumber == upload.TotalParts-1 {
		expectedSize = upload.Size - int64(partNumber)*upload.ChunkSize
	}
	if int64(len(data)) != expectedSize {
		return ErrInvalidInput
	}

	partPath := s.store.PartPath(uploadID, partNumber)
	if upload.Parts[partNumber] {
		existing, err := os.ReadFile(partPath)
		if err == nil && bytes.Equal(existing, data) {
			return nil
		}
		return ErrConflict
	}

	temporary, err := os.CreateTemp(s.store.UploadDirectory(uploadID), "part-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		os.Remove(temporaryPath)
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		os.Remove(temporaryPath)
		return err
	}
	if err := temporary.Close(); err != nil {
		os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, partPath); err != nil {
		os.Remove(temporaryPath)
		return err
	}

	upload.Parts[partNumber] = true
	return s.store.Save(upload)
}

func (s *Service) GetUpload(id string) (Upload, error) {
	upload, ok := s.store.Get(id)
	if !ok {
		return Upload{}, ErrNotFound
	}
	return upload, nil
}

func (s *Service) CompleteUpload(id string) (Upload, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.store.Get(id)
	if !ok {
		return Upload{}, ErrNotFound
	}
	if upload.Status == "ready" {
		return upload, nil
	}
	for part := 0; part < upload.TotalParts; part++ {
		if !upload.Parts[part] {
			return Upload{}, ErrConflict
		}
	}

	temporary, err := os.CreateTemp(s.store.MediaDirectory(), "upload-*.tmp")
	if err != nil {
		return Upload{}, err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	var written int64
	for part := 0; part < upload.TotalParts; part++ {
		file, err := os.Open(s.store.PartPath(id, part))
		if err != nil {
			temporary.Close()
			return Upload{}, ErrConflict
		}
		count, copyErr := io.Copy(temporary, file)
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil {
			temporary.Close()
			return Upload{}, errors.Join(copyErr, closeErr)
		}
		written += count
	}
	if written != upload.Size {
		temporary.Close()
		return Upload{}, ErrInvalidInput
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return Upload{}, err
	}
	if err := temporary.Close(); err != nil {
		return Upload{}, err
	}

	file, err := os.Open(temporaryPath)
	if err != nil {
		return Upload{}, err
	}
	configuration, format, decodeErr := image.DecodeConfig(file)
	if decodeErr != nil || configuration.Width <= 0 || configuration.Height <= 0 || configuration.Width > 10000 || configuration.Height > 10000 || int64(configuration.Width)*int64(configuration.Height) > 16_000_000 {
		file.Close()
		return Upload{}, ErrInvalidInput
	}
	contentType, extension := imageFormat(format)
	if contentType == "" {
		file.Close()
		return Upload{}, ErrInvalidInput
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		file.Close()
		return Upload{}, err
	}
	decoded, decodedFormat, err := image.Decode(file)
	file.Close()
	if err != nil || decodedFormat != format || decoded.Bounds().Dx() != configuration.Width || decoded.Bounds().Dy() != configuration.Height {
		return Upload{}, ErrInvalidInput
	}

	upload.ContentType = contentType
	upload.Extension = extension
	upload.Width = configuration.Width
	upload.Height = configuration.Height
	upload.Status = "ready"
	mediaPath := s.store.MediaPath(upload)
	if err := os.Rename(temporaryPath, mediaPath); err != nil {
		return Upload{}, err
	}
	if err := s.store.Save(upload); err != nil {
		os.Remove(mediaPath)
		return Upload{}, err
	}
	_ = os.RemoveAll(s.store.UploadDirectory(id))
	return upload, nil
}

func (s *Service) ListMedia() []Upload {
	return s.store.ListMedia()
}

func (s *Service) DeleteMedia(id string) error {
	return s.store.DeleteMedia(id)
}

func imageFormat(format string) (string, string) {
	switch format {
	case "jpeg":
		return "image/jpeg", ".jpg"
	case "png":
		return "image/png", ".png"
	case "gif":
		return "image/gif", ".gif"
	default:
		return "", ""
	}
}
