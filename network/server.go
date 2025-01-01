package network

import (
	"dfs/metadata"
	"dfs/storage"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
)

// UploadHandler 处理文件上传并进行分块（支持断点续传）
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求中读取文件名
	fileName := r.FormValue("fileName")
	if fileName == "" {
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}

	// 确定分块目录（基于文件名生成固定目录）
	chunkDir := filepath.Join("./chunks", fileName)

	// 检查元数据，获取已有的分块信息
	var existingChunks []storage.ChunkMetadata
	if meta, exists := metadata.GetMetadata(fileName); exists {
		existingChunks = meta.ChunkMetas
		fmt.Printf("Existing chunks loaded for file %s: %+v\n", fileName, existingChunks)
	} else {
		fmt.Printf("No existing chunks found for file %s\n", fileName)
	}

	// 从请求中读取文件
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 分块文件，支持断点续传
	meta, err := metadata.CreateFileMetadataFromStream(file, chunkDir, existingChunks)
	if err != nil {
		http.Error(w, "Unable to split file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回分块元数据
	response := map[string]interface{}{
		"fileName": fileName,
		"chunks":   meta.ChunkMetas,
	}
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, "Unable to encode response: "+err.Error(), http.StatusInternalServerError)
	}
}

// StartServer 启动 HTTP 服务
func StartServer(port string) {
	// 路由处理
	http.HandleFunc("/upload", UploadHandler)

	// 启动服务
	fmt.Println("Starting server on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
