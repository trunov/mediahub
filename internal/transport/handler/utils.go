package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

type APIError struct {
	Error string `json:"error"`
	Code  int    `json:"code,omitempty"`
}

func writeMultipartError(w http.ResponseWriter, err error) {
	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "too large"):
		writeJSONError(w, "uploaded file exceeds maximum allowed size", http.StatusRequestEntityTooLarge)

	case strings.Contains(msg, "content-type isn't multipart/form-data"):
		writeJSONError(w, "invalid content type, expected multipart/form-data", http.StatusBadRequest)

	default:
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseInt64Default(s string, def int64) int64 {
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return v
}

func validationErrorsToMap(err error) map[string]string {
	errs := map[string]string{}
	if verrs, ok := err.(validator.ValidationErrors); ok {
		for _, e := range verrs {
			field := e.Field()
			switch e.Tag() {
			case "required":
				errs[field] = "is required"
			case "max":
				errs[field] = "exceeds maximum length"
			case "gte", "lte":
				errs[field] = "out of allowed range"
			default:
				errs[field] = "invalid value"
			}
		}
	} else {
		errs["error"] = err.Error()
	}
	return errs
}

func writeJSONError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	_ = json.NewEncoder(w).Encode(APIError{
		Error: message,
	})
}

var allowedMIMEs = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/webp": {},
}

func validateMimeType(mimeType string) error {
	if _, ok := allowedMIMEs[mimeType]; !ok {
		return fmt.Errorf("requested file upload with invalid type: %s", mimeType)
	}
	return nil
}
