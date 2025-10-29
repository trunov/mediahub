package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-playground/validator/v10"
	"github.com/trunov/mediahub/internal/config"
	"github.com/trunov/mediahub/internal/entities"
)

type UseCase interface {
	UploadImage(ctx context.Context, file multipart.File, fh *multipart.FileHeader, ext string, fileType string, imageParams UploadImageParams) (entities.Image, error)
}

type Handler struct {
	useCase   UseCase
	cfg       *config.Config
	validator *validator.Validate
}

func New(useCase UseCase, cfg *config.Config) *Handler {
	return &Handler{
		useCase:   useCase,
		cfg:       cfg,
		validator: validator.New(),
	}
}

func (h *Handler) UploadImage(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.cfg.Upload.MaxRequestBodyMB<<20)

	maxMultipartMem := h.cfg.Upload.MaxMultipartMemoryMB
	if err := r.ParseMultipartForm(maxMultipartMem << 20); err != nil {
		writeMultipartError(w, err)
		return
	}

	file, fh, err := r.FormFile("image")
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			writeJSONError(w, `missing image file: form field key should be "image"`, http.StatusBadRequest)
		} else {
			writeJSONError(w, "an error occurred while uploading the file: "+err.Error(), http.StatusBadRequest)
		}
		return
	}
	defer file.Close()

	params := UploadImageParams{
		// should add proper validation
		ItemID:           parseInt64Default(r.Form.Get("itemID"), 0),
		SKU:              r.Form.Get("sku"),
		Context:          r.Form.Get("context"),
		Description:      r.Form.Get("description"),
		Project:          r.Form.Get("project"),
		OrderIndex:       parseInt64Default(r.Form.Get("orderIndex"), 0),
		PreserveFilename: r.URL.Query().Get("preserveFilename") == "1",
		UserID:           parseInt64Default(r.Form.Get("userID"), 0),
	}

	if err := h.validator.Struct(params); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(validationErrorsToMap(err))
		return
	}

	mime, err := mimetype.DetectReader(file)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = file.Seek(0, 0)
	if err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ext := mime.Extension()
	fileType := mime.String()

	if err := validateMimeType(fileType); err != nil {
		writeJSONError(w, fmt.Sprintf("unsupported file type: %s", fileType), http.StatusBadRequest)
		return
	}

	ctx := context.Background()

	img, err := h.useCase.UploadImage(ctx, file, fh, ext, fileType, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(img); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
