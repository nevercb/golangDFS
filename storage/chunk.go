package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const ChunkSize = 1024 * 1024 // 每块大小为 1MB

// ChunkMetadata 保存每个分块的路径和校验和
type ChunkMetadata struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	ChunkSize int64  `json:"chunk_size"`
}

// SplitFileFromStream 从文件流分块，并支持断点续传
func SplitFileFromStream(file io.Reader, destDir string, existingChunks []ChunkMetadata) ([]ChunkMetadata, error) {
	// 确保分块目录存在
	err := os.MkdirAll(destDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating chunk directory: %w", err)
	}

	var chunks []ChunkMetadata
	buffer := make([]byte, ChunkSize)
	index := 0
	skippedChunks := 0
	regeneratedChunks := 0

	for {
		// 读取分块数据
		n, err := file.Read(buffer)
		if n > 0 {
			// 计算当前分块校验和
			chunkHash := sha256.Sum256(buffer[:n])
			checksum := hex.EncodeToString(chunkHash[:])

			// 检查是否已有相同的分块
			var existingChunk *ChunkMetadata
			if index < len(existingChunks) {
				existingChunk = &existingChunks[index]
			}

			if existingChunk != nil && existingChunk.Checksum == checksum && existingChunk.ChunkSize == int64(n) {
				if _, err := os.Stat(existingChunk.Path); os.IsNotExist(err) {
					fmt.Printf("Chunk file missing, regenerating: %s\n", existingChunk.Path)
				} else {
					fmt.Printf("Skipping chunk %d: %s\n", index, existingChunk.Path)
					chunks = append(chunks, *existingChunk)
					skippedChunks++
					index++
					continue
				}
			}

			// 否则，重新生成分块
			chunkFileName := fmt.Sprintf("chunk_%d", index)
			chunkPath := filepath.Join(destDir, chunkFileName)
			chunkFile, err := os.Create(chunkPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create chunk file: %w", err)
			}
			_, writeErr := chunkFile.Write(buffer[:n])
			chunkFile.Close()
			if writeErr != nil {
				return nil, fmt.Errorf("failed to write chunk file: %w", writeErr)
			}

			// 保存新分块元数据
			chunks = append(chunks, ChunkMetadata{
				Path:      chunkPath,
				Checksum:  checksum,
				ChunkSize: int64(n),
			})
			regeneratedChunks++
			fmt.Printf("Regenerated chunk %d: %s\n", index, chunkPath)
			index++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading file: %w", err)
		}
	}

	// 打印跳过和重新生成的分块统计
	fmt.Printf("File split complete: %d chunks skipped, %d chunks regenerated\n", skippedChunks, regeneratedChunks)
	return chunks, nil
}

// MergeChunks 将多个块合并为一个文件，并验证分块的校验和
func MergeChunks(chunks []ChunkMetadata, outputFilePath string) error {
	// 创建目标文件
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer outputFile.Close()

	// 合并每个分块
	for _, chunk := range chunks {
		chunkFile, err := os.Open(chunk.Path)
		if err != nil {
			return fmt.Errorf("error opening chunk file (%s): %w", chunk.Path, err)
		}

		// 读取分块内容
		buffer := make([]byte, chunk.ChunkSize)
		n, err := chunkFile.Read(buffer)
		chunkFile.Close() // 立即关闭分块文件
		if err != nil && err != io.EOF {
			return fmt.Errorf("error reading chunk (%s): %w", chunk.Path, err)
		}

		if int64(n) != chunk.ChunkSize {
			return fmt.Errorf("chunk size mismatch for %s", chunk.Path)
		}

		// 验证校验和
		hash := sha256.Sum256(buffer[:n])
		checksum := hex.EncodeToString(hash[:])
		if checksum != chunk.Checksum {
			return fmt.Errorf("checksum mismatch for chunk %s", chunk.Path)
		}

		// 写入分块内容到目标文件
		_, err = outputFile.Write(buffer[:n])
		if err != nil {
			return fmt.Errorf("error writing chunk (%s) to output file: %w", chunk.Path, err)
		}
	}

	return nil
}
