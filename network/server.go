package network

import (
	"dfs/storage"
	"encoding/json"
	"io"
	"net/http"
	"os"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to read file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		http.Error(w, "Unable to create temp file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}
	chunkPaths, err := storage.SplitFile(tempFile.Name(), "./chunks")
	if err != nil {
		http.Error(w, "Unable to split file", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"chunkPaths": chunkPaths,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func StartServer(port string) {
	http.HandleFunc("/upload", UploadHandler)
	http.ListenAndServe(":"+port, nil)
}
