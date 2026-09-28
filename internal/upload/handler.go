package upload

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
)

type Handler struct {
	service *Service
}

type CreateUploadRequest struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

type uploadResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	ContentType    string `json:"contentType,omitempty"`
	ChunkSize      int64  `json:"chunkSize"`
	TotalParts     int    `json:"totalParts"`
	CompletedParts []int  `json:"completedParts"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
	Width          int    `json:"width,omitempty"`
	Height         int    `json:"height,omitempty"`
	URL            string `json:"url,omitempty"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/media", h.ListMedia)
	mux.HandleFunc("POST /api/uploads", h.CreateUpload)
	mux.HandleFunc("GET /api/uploads/{id}", h.GetUpload)
	mux.HandleFunc("PUT /api/uploads/{id}/parts/{part}", h.UploadPart)
	mux.HandleFunc("POST /api/uploads/{id}/complete", h.CompleteUpload)
	mux.HandleFunc("GET /api/media/{id}/file", h.GetMediaFile)
	mux.HandleFunc("DELETE /api/media/{id}", h.DeleteMedia)
}

func (h *Handler) CreateUpload(w http.ResponseWriter, r *http.Request) {
	var request CreateUploadRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, ErrInvalidInput)
		return
	}
	upload, err := h.service.CreateUpload(request.Name, request.Size)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUploadResponse(upload))
}

func (h *Handler) GetUpload(w http.ResponseWriter, r *http.Request) {
	upload, err := h.service.GetUpload(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUploadResponse(upload))
}

func (h *Handler) UploadPart(w http.ResponseWriter, r *http.Request) {
	part, err := strconv.Atoi(r.PathValue("part"))
	if err != nil {
		writeError(w, ErrInvalidInput)
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, ChunkSize+1))
	if err != nil {
		writeError(w, ErrTooLarge)
		return
	}
	if int64(len(body)) > ChunkSize {
		writeError(w, ErrTooLarge)
		return
	}
	if err := h.service.UploadPart(r.PathValue("id"), part, body); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CompleteUpload(w http.ResponseWriter, r *http.Request) {
	upload, err := h.service.CompleteUpload(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toUploadResponse(upload))
}

func (h *Handler) ListMedia(w http.ResponseWriter, r *http.Request) {
	items := h.service.ListMedia()
	response := make([]uploadResponse, 0, len(items))
	for _, item := range items {
		response = append(response, toUploadResponse(item))
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": response})
}

func (h *Handler) GetMediaFile(w http.ResponseWriter, r *http.Request) {
	upload, err := h.service.GetUpload(r.PathValue("id"))
	if err != nil || upload.Status != "ready" {
		writeError(w, ErrNotFound)
		return
	}
	w.Header().Set("Content-Type", upload.ContentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, h.service.store.MediaPath(upload))
}

func (h *Handler) DeleteMedia(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteMedia(r.PathValue("id")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toUploadResponse(item Upload) uploadResponse {
	parts := make([]int, 0, len(item.Parts))
	for part, complete := range item.Parts {
		if complete {
			parts = append(parts, part)
		}
	}
	sort.Ints(parts)
	response := uploadResponse{
		ID:             item.ID,
		Name:           item.Name,
		Size:           item.Size,
		ContentType:    item.ContentType,
		ChunkSize:      item.ChunkSize,
		TotalParts:     item.TotalParts,
		CompletedParts: parts,
		Status:         item.Status,
		CreatedAt:      item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		Width:          item.Width,
		Height:         item.Height,
	}
	if item.Status == "ready" {
		response.URL = "/api/media/" + item.ID + "/file"
	}
	return response
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status, message := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, ErrInvalidInput):
		status, message = http.StatusBadRequest, "invalid request or unsupported image"
	case errors.Is(err, ErrTooLarge):
		status, message = http.StatusRequestEntityTooLarge, "image exceeds the 2 MiB limit"
	case errors.Is(err, ErrNotFound):
		status, message = http.StatusNotFound, "upload or image not found"
	case errors.Is(err, ErrConflict):
		status, message = http.StatusConflict, "upload is incomplete or already finalized"
	}
	writeJSON(w, status, map[string]string{"error": message})
}
