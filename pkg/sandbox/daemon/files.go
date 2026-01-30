package daemon

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type fileContent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func handleFiles(w http.ResponseWriter, r *http.Request, root string) {
	// Path after /files/
	rel := strings.TrimPrefix(r.URL.Path, "/files")
	rel = strings.TrimPrefix(rel, "/")

	fullPath, err := resolvePath(root, rel)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handleFileGet(w, fullPath, rel)
	case http.MethodPost, http.MethodPut:
		handleFileWrite(w, r, fullPath, rel)
	case http.MethodDelete:
		handleFileDelete(w, fullPath)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func handleFileGet(w http.ResponseWriter, fullPath, rel string) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"read failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(fileContent{
		Path:    rel,
		Content: string(data),
	})
}

func handleFileWrite(w http.ResponseWriter, r *http.Request, fullPath, rel string) {
	var fc fileContent
	if err := json.NewDecoder(r.Body).Decode(&fc); err != nil && err != io.EOF {
		http.Error(w, fmt.Sprintf(`{"error":"invalid json: %v"}`, err), http.StatusBadRequest)
		return
	}

	content := fc.Content

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"mkdir failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"write failed: %v"}`, err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(fileContent{
		Path:    rel,
		Content: content,
	})
}

func handleFileDelete(w http.ResponseWriter, fullPath string) {
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf(`{"error":"delete failed: %v"}`, err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
