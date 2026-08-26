# 本仓评测钉 go 1.21-alpine（与 newtask-builder / 出题容器一致）
FROM golang:1.21-alpine
ENV GOTOOLCHAIN=local
ENV CGO_ENABLED=0

WORKDIR /src

# 先复制依赖文件并下载依赖（利用 Docker 缓存，也保证容器内离线可用）
COPY go.mod go.sum ./
RUN go mod download

# 复制所有项目文件
COPY . .

# 预编译服务二进制；模型仍可改源码再 go build
RUN go build -o /app/bin/server .

EXPOSE 8080
# 云质检 docker run -d -P：CMD 必须是常驻 HTTP，不能是 /bin/sh
CMD ["/app/bin/server", "serve"]
