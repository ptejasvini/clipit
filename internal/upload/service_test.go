package upload

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestUploadLifecyclePersistsAndDeletesMedia(t *testing.T) {
	root := t.TempDir()
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)

	var imageData bytes.Buffer
	imageValue := image.NewRGBA(image.Rect(0, 0, 2, 1))
	imageValue.Set(0, 0, color.RGBA{R: 255, A: 255})
	imageValue.Set(1, 0, color.RGBA{G: 255, A: 255})
	if err := png.Encode(&imageData, imageValue); err != nil {
		t.Fatal(err)
	}

	created, err := service.CreateUpload("sample.png", int64(imageData.Len()))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.UploadPart(created.ID, 0, imageData.Bytes()); err != nil {
		t.Fatal(err)
	}
	completed, err := service.CompleteUpload(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != "ready" || completed.Width != 2 || completed.Height != 1 {
		t.Fatalf("unexpected completed upload: %+v", completed)
	}

	reopened, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if media := reopened.ListMedia(); len(media) != 1 || media[0].ID != created.ID {
		t.Fatalf("expected persisted media, got %+v", media)
	}
	if err := reopened.DeleteMedia(created.ID); err != nil {
		t.Fatal(err)
	}
	if media := reopened.ListMedia(); len(media) != 0 {
		t.Fatalf("expected media deletion, got %+v", media)
	}
}

func TestCreateUploadRejectsInvalidSizes(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	for _, size := range []int64{0, -1, MaxFileSize + 1} {
		if _, err := service.CreateUpload("sample.png", size); err == nil {
			t.Fatalf("expected size %d to be rejected", size)
		}
	}
}

func TestCompleteUploadRejectsMalformedImage(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := NewService(store)
	data := []byte("not an image")
	upload, err := service.CreateUpload("broken.png", int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.UploadPart(upload.ID, 0, data); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompleteUpload(upload.ID); err != ErrInvalidInput {
		t.Fatalf("expected malformed image rejection, got %v", err)
	}
}
