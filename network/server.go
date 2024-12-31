package network

import (
	"dfs/storage"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// UploadHandler 处理文件上传并进行分块
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求中读取文件
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 创建临时文件
	tempFile, err := os.CreateTemp("", "upload-*.tmp")
	if err != nil {
		http.Error(w, "Unable to create temp file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() {
		tempFile.Close()
		os.Remove(tempFile.Name()) // 删除临时文件
	}()

	// 将上传的文件保存到临时文件
	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Unable to save file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 分块文件
	chunks, err := storage.SplitFile(tempFile.Name(), "./chunks")
	if err != nil {
		http.Error(w, "Unable to split file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 返回分块元数据
	response := map[string]interface{}{
		"chunks": chunks,
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
