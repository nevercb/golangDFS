package metadata

type FileMetadata struct {
	FileName    string   `json:"file_name"`
	ChunkPaths  []string `json:"chunk_paths"`
	ChunkHashes []string `json:"chunk_hashes"`
}

// MetadataMap 管理所有文件的元数据
var MetadataMap = make(map[string]FileMetadata)

// AddMetadata 新增文件元数据
func AddMetadata(fileName string, metadata FileMetadata) {
	MetadataMap[fileName] = metadata
}

// GetMetadata 获取文件元数据
func GetMetadata(fileName string) (FileMetadata, bool) {
	metadata, exists := MetadataMap[fileName]
	return metadata, exists
}
