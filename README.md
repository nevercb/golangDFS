# Golang Distrubuted File System

## Functionality:
  * spliting file into chunks, uploading chunks.
  * merging file chunks.
  * metadata management(add, save, load).
  * File chunks hash verification.
  * resumable upload (both server and client side)
  * ...
## Pending to do
  * 文件秒传
  * 分布式多节点存储
  * 支持客户端Range请求
  * ....

## test  
* 测试完整文件上传

选择菜单选项 1，上传整个文件并分块：

Choose an option:
1.Upload and split file (stream)
...
Enter your choice: 1
输入要上传的文件路径，例如：
Enter the file path to upload: /path/to/testfile
服务端会将文件分块并保存到 ./chunks/testfile/ 目录下，同时会记录分块元数据。
检查服务端是否生成了分块文件和元数据，您可以在 ./chunks/ 目录中看到分块文件，例如：
./chunks/testfile/chunk_0
./chunks/testfile/chunk_1
...
* 使用选项 3 查看元数据

Metadata: 
File: testfile
Metadata: {FileName:testfile ChunkMetas:[{Path:./chunks/testfile/chunk_0 Checksum:... ChunkSize:...} ...]}

* 测试客户端断点续传

手动删除服务端的部分分块。例如，删除 chunk_1 和 chunk_3：
rm ./chunks/testfile/chunk_1
rm ./chunks/testfile/chunk_3
检查分块是否确实被删除。
* 客户端调用断点续传功能

选择菜单选项 5，测试客户端的断点续传功能：
Choose an option:
5. Test client-side resumable upload
...
Enter your choice: 5
* 输入测试文件路径和服务端地址

Enter the file path to upload: /path/to/testfile
Enter the server URL (e.g., http://localhost:8080): http://localhost:8080
客户端会调用 /checkChunks 接口，获取服务端已有的分块信息。
客户端会自动对比本地文件的分块校验和，重新上传丢失的分块。例如：
Existing chunks for file testfile: [{Path:./chunks/testfile/chunk_0 Checksum:... ChunkSize:...} ...]
Uploading chunk 1
Uploading chunk 3
* 检查服务端是否重新生成了丢失的分块文件。
