package upload

type Store struct {
	uploads map[string]Upload
}

func NewStore() *Store {
	return &Store{
		uploads: make(map[string]Upload),
	}
}

func (s *Store) Save(upload Upload) {
	s.uploads[upload.ID] = upload
}

func (s *Store) Get(id string) (Upload, bool) {
	upload, ok := s.uploads[id]

	return upload, ok
}
