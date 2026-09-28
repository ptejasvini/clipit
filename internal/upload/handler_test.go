package upload

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestMediaHTTPWorkflow(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	NewHandler(NewService(store)).RegisterRoutes(mux)

	var imageData bytes.Buffer
	imageValue := image.NewRGBA(image.Rect(0, 0, 2, 1))
	imageValue.Set(0, 0, color.RGBA{R: 255, A: 255})
	imageValue.Set(1, 0, color.RGBA{G: 255, A: 255})
	if err := png.Encode(&imageData, imageValue); err != nil {
		t.Fatal(err)
	}

	create := httptest.NewRequest(http.MethodPost, "/api/uploads", strings.NewReader(`{"name":"sample.png","size":`+strconv.Itoa(imageData.Len())+`}`))
	createdResponse := httptest.NewRecorder()
	mux.ServeHTTP(createdResponse, create)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createdResponse.Code, createdResponse.Body)
	}
	var session uploadResponse
	if err := json.NewDecoder(createdResponse.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}

	part := httptest.NewRequest(http.MethodPut, "/api/uploads/"+session.ID+"/parts/0", bytes.NewReader(imageData.Bytes()))
	partResponse := httptest.NewRecorder()
	mux.ServeHTTP(partResponse, part)
	if partResponse.Code != http.StatusNoContent {
		t.Fatalf("part status = %d, body = %s", partResponse.Code, partResponse.Body)
	}

	complete := httptest.NewRequest(http.MethodPost, "/api/uploads/"+session.ID+"/complete", nil)
	completeResponse := httptest.NewRecorder()
	mux.ServeHTTP(completeResponse, complete)
	if completeResponse.Code != http.StatusOK {
		t.Fatalf("complete status = %d, body = %s", completeResponse.Code, completeResponse.Body)
	}

	list := httptest.NewRequest(http.MethodGet, "/api/media", nil)
	listResponse := httptest.NewRecorder()
	mux.ServeHTTP(listResponse, list)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), session.ID) {
		t.Fatalf("list status = %d, body = %s", listResponse.Code, listResponse.Body)
	}

	media := httptest.NewRequest(http.MethodGet, "/api/media/"+session.ID+"/file", nil)
	mediaResponse := httptest.NewRecorder()
	mux.ServeHTTP(mediaResponse, media)
	if mediaResponse.Code != http.StatusOK || mediaResponse.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("media status = %d, content type = %s", mediaResponse.Code, mediaResponse.Header().Get("Content-Type"))
	}

	remove := httptest.NewRequest(http.MethodDelete, "/api/media/"+session.ID, nil)
	removeResponse := httptest.NewRecorder()
	mux.ServeHTTP(removeResponse, remove)
	if removeResponse.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", removeResponse.Code, removeResponse.Body)
	}
}
