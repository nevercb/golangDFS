package main

import (
	"dfs/metadata"
	"fmt"
	"os"
)

func main() {
	// 服务启动时加载元数据
	err := metadata.LoadMetadata()
	if err != nil {
		fmt.Println("Error loading metadata:", err)
		return
	}
	fmt.Println("Metadata loaded successfully.")

	// 提供用户操作选项
	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. Upload and split file")
		fmt.Println("2. Merge file")
		fmt.Println("3. Show metadata")
		fmt.Println("4. Save metadata to file")
		fmt.Println("5. Exit")

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
		fmt.Println("File does not exist:", filePath)
		return
	}

	// 分块并生成元数据
	metadata, err := metadata.CreateFileMetadata(filePath, "./chunks")
	if err != nil {
		fmt.Println("Error splitting file:", err)
		return
	}

	fmt.Println("File uploaded and split successfully.")
	fmt.Printf("Metadata: %+v\n", metadata)
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
