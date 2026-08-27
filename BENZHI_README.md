Go 写的 Reed-Solomon 命令行：encode 把文件切成数据片加 GF(2^8) 校验片并写下 meta.json，reconstruct 用还在的分片做高斯消元重建；serve 把分片目录做成 HTTP 服务。

# erasure-code

Reed-Solomon erasure coding service with file-based shard storage, parity
verification, and data reconstruction from degraded shard sets.

## Build / Run / Test

```bash
go build -o erasure-code .
./erasure-code serve -addr :8080 -dir ./shards
go test ./...
```

## Evaluation Image

Evaluation-specific files (do not overwrite project Dockerfile/README):

- `benzhi.Dockerfile`
- `build_benzhi_docker.sh`
- `BENZHI_README.md` (this file)

Build and verify in container:

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh <image-name> linux/arm64
./build_benzhi_docker.sh <image-name> linux/amd64
docker run -it <image-name>:latest
```
