package storage

import (
	"io"
	"os"
	"strconv"
)

const ChunkSize = 1024 * 1024

func SplitFile(filePath string, destDir string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var chunkPaths []string
	buffer := make([]byte, ChunkSize)
	index := 0

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunkPath := destDir + "/chunk_" + strconv.Itoa(index)
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
