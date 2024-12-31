package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const ChunkSize = 1024 * 1024 // 每块大小为 1MB

// ChunkMetadata 保存每个分块的路径和校验和
type ChunkMetadata struct {
	Path      string `json:"path"`
	Checksum  string `json:"checksum"`
	ChunkSize int64  `json:"chunk_size"`
}

func SplitFile(filePath string, destDir string) ([]ChunkMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	uniqueDir := filepath.Join(destDir, fmt.Sprintf("%d", time.Now().UnixNano()))
	err = os.MkdirAll(uniqueDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating unique directory: %w", err)
	}

	var chunks []ChunkMetadata
	buffer := make([]byte, ChunkSize)
	index := 0

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			// 创建唯一的分块文件
			chunkFileName := fmt.Sprintf("chunk_%d", index)
			chunkPath := filepath.Join(uniqueDir, chunkFileName)
			chunkFile, err := os.Create(chunkPath)
			if err != nil {
				return nil, err
			}
			_, writeErr := chunkFile.Write(buffer[:n])
			defer chunkFile.Close()
			if writeErr != nil {
				return nil, writeErr
			}
			// 计算分块的校验和
			hash := sha256.Sum256(buffer[:n])
			checksum := hex.EncodeToString(hash[:])

			// 保存分块元数据
			chunks = append(chunks, ChunkMetadata{
				Path:      chunkPath,
				Checksum:  checksum,
				ChunkSize: int64(n),
			})
			index++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
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
