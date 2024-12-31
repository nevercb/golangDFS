package metadata

import (
	"dfs/storage"
	"encoding/json"
	"os"
	"sync"
)

// FileMetadata 保存文件的元数据，包括文件名、分块元数据等
type FileMetadata struct {
	FileName   string                  `json:"file_name"`
	ChunkMetas []storage.ChunkMetadata `json:"chunk_metas"` // 分块元数据
}

// MetadataMap 管理所有文件的元数据（线程安全）
var (
	MetadataMap = make(map[string]FileMetadata) // 文件名到元数据的映射
	mutex       sync.RWMutex                    // 读写锁，保证线程安全
)

// MetadataFile 文件元数据的持久化存储路径
const MetadataFile = "metadata.json"

// AddMetadata 新增文件元数据
func AddMetadata(fileName string, metadata FileMetadata) {
	mutex.Lock()
	defer mutex.Unlock()
	MetadataMap[fileName] = metadata
}

// GetMetadata 获取文件元数据
func GetMetadata(fileName string) (FileMetadata, bool) {
	mutex.RLock()
	defer mutex.RUnlock()
	metadata, exists := MetadataMap[fileName]
	return metadata, exists
}

// DeleteMetadata 删除文件元数据
func DeleteMetadata(fileName string) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(MetadataMap, fileName)
}

// SaveMetadata 将元数据保存到文件
func SaveMetadata() error {
	mutex.RLock()
	defer mutex.RUnlock()

	// 将 MetadataMap 转换为 JSON
	data, err := json.MarshalIndent(MetadataMap, "", "  ")
	if err != nil {
		return err
	}

	// 写入文件
	return os.WriteFile(MetadataFile, data, 0644)
}

// LoadMetadata 从文件加载元数据
func LoadMetadata() error {
	mutex.Lock()
	defer mutex.Unlock()

	// 读取文件
	data, err := os.ReadFile(MetadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 文件不存在，返回空映射
		}
		return err
	}

	// 将 JSON 数据解析为 MetadataMap
	return json.Unmarshal(data, &MetadataMap)
}

// CreateFileMetadata 使用 storage 包的 SplitFile，将文件分块并生成元数据
func CreateFileMetadata(filePath, destDir string) (FileMetadata, error) {
	// 调用 storage.SplitFile 分块文件
	chunks, err := storage.SplitFile(filePath, destDir)
	if err != nil {
		return FileMetadata{}, err
	}

	// 创建 FileMetadata
	metadata := FileMetadata{
		FileName:   filePath,
		ChunkMetas: chunks,
	}

	// 添加到 MetadataMap
	AddMetadata(filePath, metadata)

	return metadata, nil
}

// MergeFile 根据文件名从元数据中读取信息，并合并分块文件
func MergeFile(fileName, outputFilePath string) error {
	// 获取文件元数据
	metadata, exists := GetMetadata(fileName)
	if !exists {
		return os.ErrNotExist
	}

	// 调用 storage.MergeChunks 合并文件
	return storage.MergeChunks(metadata.ChunkMetas, outputFilePath)
}
