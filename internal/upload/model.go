package upload

type Upload struct {
	ID         string
	Size       int64
	ChunkSize  int64
	TotalParts int
	UploadPart map[int][]byte
}
