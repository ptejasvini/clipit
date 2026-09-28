package upload

import "time"

type Upload struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Size        int64        `json:"size"`
	ContentType string       `json:"contentType,omitempty"`
	ChunkSize   int64        `json:"chunkSize"`
	TotalParts  int          `json:"totalParts"`
	Parts       map[int]bool `json:"parts"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"createdAt"`
	Width       int          `json:"width,omitempty"`
	Height      int          `json:"height,omitempty"`
	Extension   string       `json:"extension,omitempty"`
}
