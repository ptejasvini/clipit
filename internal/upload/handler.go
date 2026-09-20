package upload

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	service *Service
}

type CreateUploadRequest struct {
	Size int64 `json:"size"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}
func (h *Handler) CreateUpload(w http.ResponseWriter, r *http.Request) {
	var request CreateUploadRequest

	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, "Invalid Request Body ", http.StatusBadRequest)
		return
	}
	upload := h.service.CreateUpload(request.Size)
	response := map[string]interface{}{
		"uploadId": upload.ID,
		"Size":     upload.Size,
		"ChunkSize": upload.ChunkSize,
		"totalParts": upload.TotalParts,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
