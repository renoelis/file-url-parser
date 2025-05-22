FROM golang:1.24-alpine AS builder

WORKDIR /app

# 设置Go代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GO111MODULE=on

COPY . .

# 添加重试机制
RUN go mod tidy || go mod tidy && go build -o main ./cmd/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/config ./config

EXPOSE 4001

# 以下环境变量定义已移动到docker-compose文件中，避免在镜像中嵌入配置
# ENV PORT=4001
# ENV PYTHON_SERVICE_URL=http://file-url-parser-python:4002
# ENV MAX_FILE_SIZE=10485760
# ENV MAX_ALLOWED_ROWS=200
# ENV USE_HEADER_AS_KEY=true

CMD ["./main"] 