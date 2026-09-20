package upload

import (
	"github.com/google/uuid"
	"fmt"
)

const ChunkSize int64 = 512 * 1024

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) CreateUpload(size int64) Upload {
	totalParts := int((size + ChunkSize - 1) / ChunkSize)

	upload := Upload{
		ID:         uuid.NewString(),
		TotalParts: totalParts,
		ChunkSize:  ChunkSize,
		Size : size,
		UploadPart: make(map[int][]byte),
	}
	s.store.Save(upload)
	return upload
}

func (s *Service) UploadPart(uploadID string, partNumber int, data []byte) error {
	upload, ok := s.store.Get(uploadID)
	if !ok {
		return fmt.Errorf("upload not found")
	}

	if partNumber < 0 || partNumber >= upload.TotalParts {
		return fmt.Errorf("invalid part number")
	}

	upload.UploadedPart[partNumber] = data

	s.store.Save(upload)

	return nil
}
