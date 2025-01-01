package network

import (
	"crypto/sha256"
	"dfs/metadata"
	"dfs/storage"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func UploadHandler(w http.ResponseWriter, r *http.Request) {
	// 从请求中读取文件名和分块索引
	fileName := r.FormValue("fileName")
	if fileName == "" {
		http.Error(w, "File name is required", http.StatusBadRequest)
		return
	}

	chunkIndexStr := r.FormValue("chunkIndex")
	if chunkIndexStr == "" {
		http.Error(w, "Chunk index is required", http.StatusBadRequest)
		return
	}
	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil {
		http.Error(w, "Invalid chunk index", http.StatusBadRequest)
		return
	}

	chunkChecksum := r.FormValue("checksum")
	if chunkChecksum == "" {
		http.Error(w, "Checksum is required", http.StatusBadRequest)
		return
	}

	// 确定分块目录（基于文件名生成固定目录）
	chunkDir := filepath.Join("./chunks", fileName)

	// 确保分块目录存在
	err = os.MkdirAll(chunkDir, os.ModePerm)
	if err != nil {
		http.Error(w, "Unable to create chunk directory: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 从请求中读取分块文件
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Unable to read file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 写入分块文件
	chunkFileName := fmt.Sprintf("chunk_%d", chunkIndex)
	chunkPath := filepath.Join(chunkDir, chunkFileName)
	chunkFile, err := os.Create(chunkPath)
	if err != nil {
		http.Error(w, "Unable to create chunk file: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer chunkFile.Close()

	// 读取并写入分块数据
	_, err = io.Copy(chunkFile, file)
	if err != nil {
		http.Error(w, "Unable to write chunk file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 校验分块完整性（计算校验和）
	chunkFile.Seek(0, io.SeekStart) // 重置文件指针到开头
	hash := sha256.New()
	_, err = io.Copy(hash, chunkFile)
	if err != nil {
		http.Error(w, "Unable to calculate checksum: "+err.Error(), http.StatusInternalServerError)
		return
	}
	calculatedChecksum := hex.EncodeToString(hash.Sum(nil))
	if chunkChecksum != calculatedChecksum {
		http.Error(w, "Checksum mismatch", http.StatusBadRequest)
		return
	}

	// 更新元数据
	meta, exists := metadata.GetMetadata(fileName)
	if !exists {
		// 如果元数据不存在，则创建新的元数据
		meta = metadata.FileMetadata{
			FileName:   fileName,
			ChunkMetas: []storage.ChunkMetadata{},
		}
	}

	// 检查分块是否已存在元数据中，防止重复添加
	isChunkExists := false
	for _, chunk := range meta.ChunkMetas {
		if chunk.Path == chunkPath {
			isChunkExists = true
			break
		}
	}

	if !isChunkExists {
		// 添加新分块到元数据
		newChunk := storage.ChunkMetadata{
			Path:      chunkPath,
			Checksum:  calculatedChecksum,
			ChunkSize: int64(hash.Size()),
		}
		meta.ChunkMetas = append(meta.ChunkMetas, newChunk)

		// 保存更新后的元数据
		metadata.AddMetadata(fileName, meta)
		err := metadata.SaveMetadata()
		if err != nil {
			http.Error(w, "Failed to save metadata: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 返回成功响应
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Chunk %d uploaded and metadata updated successfully.", chunkIndex)
}

func CheckChunksHandler(w http.ResponseWriter, r *http.Request) {
	// 获取文件名
	fileName := r.URL.Query().Get("fileName")
	if fileName == "" {
		http.Error(w, "Missing fileName parameter", http.StatusBadRequest)
		return
	}

	// 从元数据中获取文件的所有分块信息
	meta, exists := metadata.GetMetadata(fileName)
	if !exists {
		http.Error(w, "File metadata not found", http.StatusNotFound)
		return
	}

	// 检查每个分块是否存在，更新元数据
	updatedChunks := []storage.ChunkMetadata{}
	for _, chunk := range meta.ChunkMetas {
		if _, err := os.Stat(chunk.Path); os.IsNotExist(err) {
			// 分块文件不存在，跳过
			fmt.Printf("Chunk missing: %s\n", chunk.Path)
		} else {
			// 分块文件存在，保留在元数据中
			updatedChunks = append(updatedChunks, chunk)
		}
	}

	// 如果元数据有更新，则保存
	if len(updatedChunks) != len(meta.ChunkMetas) {
		fmt.Printf("Updating metadata for file: %s\n", fileName)
		meta.ChunkMetas = updatedChunks
		metadata.AddMetadata(fileName, meta)
		metadata.SaveMetadata()
	}

	// 返回现有的分块信息
	response := map[string]interface{}{
		"fileName": fileName,
		"chunks":   updatedChunks,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func CheckChunkHandler(w http.ResponseWriter, r *http.Request) {
	// 获取请求参数
	fileName := r.URL.Query().Get("fileName")
	chunkIndex := r.URL.Query().Get("chunkIndex")
	checksum := r.URL.Query().Get("checksum")

	// 校验参数
	if fileName == "" || chunkIndex == "" || checksum == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	// 确定分块路径
	chunkDir := filepath.Join("./chunks", fileName)
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%s", chunkIndex))

	// 检查分块文件是否存在
	if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
		// 文件不存在，返回 false
		response := map[string]bool{"exists": false}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 校验分块的校验和
	file, err := os.Open(chunkPath)
	if err != nil {
		http.Error(w, "Error opening chunk file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		http.Error(w, "Error calculating checksum", http.StatusInternalServerError)
		return
	}
	calculatedChecksum := hex.EncodeToString(hash.Sum(nil))

	// 返回分块是否存在（校验和匹配）
	exists := (calculatedChecksum == checksum)
	response := map[string]bool{"exists": exists}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StartServer 启动 HTTP 服务
func StartServer(port string) {
	// 路由处理
	http.HandleFunc("/upload", UploadHandler)
	http.HandleFunc("/checkChunks", CheckChunksHandler)
	http.HandleFunc("/checkChunk", CheckChunkHandler)

	// 启动服务
	fmt.Println("Starting server on port", port)
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
