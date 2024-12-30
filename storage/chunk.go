package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const ChunkSize = 1024 * 1024

func SplitFile(filePath string, destDir string) ([]string, error) {
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

	var chunkPaths []string
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
			chunkPaths = append(chunkPaths, chunkPath)
			index++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return chunkPaths, nil
}

// MergeChunks 将多个块合并成一个文件

func MergeChunks(chunkPaths []string, outputFilePath string) error {
	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	for _, chunkPath := range chunkPaths {
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return err
		}
		defer chunkFile.Close()

		_, err = io.Copy(outputFile, chunkFile)
		if err != nil {
			return err
		}
	}
	return nil
}
