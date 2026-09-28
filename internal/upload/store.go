package upload

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

type Store struct {
	mu      sync.RWMutex
	root    string
	uploads map[string]Upload
}

func NewStore(root string) (*Store, error) {
	if root == "" {
		root = "data"
	}
	for _, directory := range []string{root, filepath.Join(root, "uploads"), filepath.Join(root, "media")} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return nil, err
		}
	}

	store := &Store{root: root, uploads: make(map[string]Upload)}
	data, err := os.ReadFile(filepath.Join(root, "uploads.json"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		if err := json.Unmarshal(data, &store.uploads); err != nil {
			return nil, err
		}
	}
	if store.uploads == nil {
		store.uploads = make(map[string]Upload)
	}
	return store, nil
}

func (s *Store) Save(upload Upload) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	updated := cloneUploads(s.uploads)
	updated[upload.ID] = cloneUpload(upload)
	if err := s.persist(updated); err != nil {
		return err
	}
	s.uploads = updated
	return nil
}

func (s *Store) Get(id string) (Upload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	upload, ok := s.uploads[id]
	return cloneUpload(upload), ok
}

func (s *Store) ListMedia() []Upload {
	s.mu.RLock()
	defer s.mu.RUnlock()

	media := make([]Upload, 0, len(s.uploads))
	for _, item := range s.uploads {
		if item.Status == "ready" {
			media = append(media, cloneUpload(item))
		}
	}
	sort.Slice(media, func(i, j int) bool { return media[i].CreatedAt.After(media[j].CreatedAt) })
	return media
}

func (s *Store) DeleteMedia(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.uploads[id]
	if !ok || item.Status != "ready" {
		return ErrNotFound
	}
	updated := cloneUploads(s.uploads)
	delete(updated, id)
	if err := s.persist(updated); err != nil {
		return err
	}
	s.uploads = updated
	_ = os.Remove(filepath.Join(s.root, "media", id+item.Extension))
	_ = os.RemoveAll(filepath.Join(s.root, "uploads", id))
	return nil
}

func (s *Store) PartPath(id string, part int) string {
	return filepath.Join(s.root, "uploads", id, strconv.Itoa(part)+".part")
}

func (s *Store) MediaPath(item Upload) string {
	return filepath.Join(s.root, "media", item.ID+item.Extension)
}

func (s *Store) UploadDirectory(id string) string {
	return filepath.Join(s.root, "uploads", id)
}

func (s *Store) MediaDirectory() string {
	return filepath.Join(s.root, "media")
}

func (s *Store) persist(uploads map[string]Upload) error {
	file, err := os.CreateTemp(s.root, "uploads-*.json")
	if err != nil {
		return err
	}
	temporaryPath := file.Name()
	defer os.Remove(temporaryPath)

	if err := json.NewEncoder(file).Encode(uploads); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryPath, filepath.Join(s.root, "uploads.json"))
}

func cloneUploads(source map[string]Upload) map[string]Upload {
	cloned := make(map[string]Upload, len(source))
	for id, item := range source {
		cloned[id] = cloneUpload(item)
	}
	return cloned
}

func cloneUpload(item Upload) Upload {
	parts := make(map[int]bool, len(item.Parts))
	for part, complete := range item.Parts {
		parts[part] = complete
	}
	item.Parts = parts
	return item
}
