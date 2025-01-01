package main

import (
	"bytes"
	"crypto/sha256"
	"dfs/metadata"
	"dfs/network"
	"dfs/storage"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// 服务启动时加载元数据
	err := metadata.LoadMetadata()
	if err != nil {
		fmt.Println("Error loading metadata:", err)
		return
	}
	fmt.Println("Metadata loaded successfully.")

	// 启动 HTTP 服务
	go func() {
		network.StartServer("8080")
	}()

	// 提供用户操作选项
	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Upload and split file (stream)")
		fmt.Println("2. Merge file")
		fmt.Println("3. Show metadata")
		fmt.Println("4. Save metadata to file")
		fmt.Println("5. Test client-side resumable upload")
		fmt.Println("6. Exit")

		var choice int
		fmt.Print("Enter your choice: ")
		fmt.Scan(&choice)

		switch choice {
		case 1:
			uploadAndSplitFile()
		case 2:
			mergeFile()
		case 3:
			showMetadata()
		case 4:
			saveMetadata()
		case 5:
			testResumableUpload()
		case 6:
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

func uploadAndSplitFile() {
	var filePath string
	fmt.Print("Enter the file path to upload: ")
	fmt.Scan(&filePath)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("File does not exist: %s\n", filePath)
		return
	}

	// 打开文件作为流
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	// 确定分块目录
	fileName := filepath.Base(filePath)
	chunkDir := filepath.Join("./chunks", fileName)

	// 使用流式分块逻辑
	meta, err := metadata.CreateFileMetadataFromStream(file, chunkDir, nil)
	if err != nil {
		fmt.Println("Error splitting file:", err)
		return
	}

	fmt.Println("File uploaded and split successfully.")
	fmt.Printf("Metadata: %+v\n", meta)
}

func mergeFile() {
	var fileName, outputFilePath string
	fmt.Print("Enter the file name to merge: ")
	fmt.Scan(&fileName)
	fmt.Print("Enter the output file path: ")
	fmt.Scan(&outputFilePath)

	// 合并文件
	err := metadata.MergeFile(fileName, outputFilePath)
	if err != nil {
		fmt.Println("Error merging file:", err)
		return
	}

	fmt.Println("File merged successfully:", outputFilePath)
}

func showMetadata() {
	fmt.Println("Current Metadata:")
	for fileName, meta := range metadata.MetadataMap {
		fmt.Printf("File: %s\nMetadata: %+v\n", fileName, meta)
	}
}

func saveMetadata() {
	// 保存元数据到文件
	err := metadata.SaveMetadata()
	if err != nil {
		fmt.Println("Error saving metadata:", err)
		return
	}

	fmt.Println("Metadata saved successfully.")
}

func testResumableUpload() {
	var filePath, serverURL string
	fmt.Print("Enter the file path to upload: ")
	fmt.Scan(&filePath)
	fmt.Print("Enter the server URL (e.g., http://localhost:8080): ")
	fmt.Scan(&serverURL)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Printf("File does not exist: %s\n", filePath)
		return
	}

	// 执行断点续传测试
	err := resumableUploadTest(serverURL, filePath)
	if err != nil {
		fmt.Println("Error during resumable upload:", err)
	} else {
		fmt.Println("Resumable upload completed successfully.")
	}
}

func resumableUploadTest(serverURL, filePath string) error {
	fileName := filepath.Base(filePath)

	// 查询已有分块
	existingChunks, err := getExistingChunks(serverURL, fileName)
	if err != nil {
		return fmt.Errorf("error fetching existing chunks: %w", err)
	}
	fmt.Printf("Existing chunks for file %s: %+v\n", fileName, existingChunks)

	// 创建已有分块的索引
	existingMap := make(map[string]bool)
	for _, chunk := range existingChunks {
		existingMap[chunk.Checksum] = true
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// 分块并上传
	buffer := make([]byte, storage.ChunkSize)
	index := 0
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			// 计算分块校验和
			hash := sha256.Sum256(buffer[:n])
			checksum := hex.EncodeToString(hash[:])

			// 检查是否已存在
			if existingMap[checksum] {
				fmt.Printf("Skipping chunk %d (already exists on server)\n", index)
			} else {
				fmt.Printf("Uploading chunk %d\n", index)
				err := uploadChunk(serverURL, fileName, index, buffer[:n], checksum)
				if err != nil {
					return fmt.Errorf("error uploading chunk %d: %w", index, err)
				}
			}
			index++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading file: %w", err)
		}
	}

	// 再次查询元数据以验证更新
	finalChunks, err := getExistingChunks(serverURL, fileName)
	if err != nil {
		return fmt.Errorf("error validating metadata after upload: %w", err)
	}
	fmt.Printf("Final chunks for file %s: %+v\n", fileName, finalChunks)

	return nil
}

func getExistingChunks(serverURL, fileName string) ([]storage.ChunkMetadata, error) {
	url := fmt.Sprintf("%s/checkChunks?fileName=%s", serverURL, fileName)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing chunks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status: %s", resp.Status)
	}

	var response struct {
		FileName string                  `json:"fileName"`
		Chunks   []storage.ChunkMetadata `json:"chunks"`
	}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, fmt.Errorf("failed to decode server response: %w", err)
	}

	return response.Chunks, nil
}

func uploadChunk(serverURL, fileName string, chunkIndex int, chunkData []byte, checksum string) error {
	url := fmt.Sprintf("%s/upload", serverURL)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加表单字段
	writer.WriteField("fileName", fileName)
	writer.WriteField("chunkIndex", fmt.Sprintf("%d", chunkIndex))
	writer.WriteField("checksum", checksum)

	// 添加文件内容
	part, err := writer.CreateFormFile("file", fmt.Sprintf("chunk_%d", chunkIndex))
	if err != nil {
		return err
	}
	part.Write(chunkData)
	writer.Close()

	// 发送请求
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status: %s", resp.Status)
	}

	return nil
}
