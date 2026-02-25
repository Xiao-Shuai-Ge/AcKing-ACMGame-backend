#!/bin/bash

# 定义输出文件名
OUTPUT_NAME="tgwp-linux-amd64"

echo "正在编译项目为 Linux 可执行文件..."

# 执行编译命令，并在同一行设置环境变量，这仅对该命令生效
# -ldflags="-s -w" 用于减小二进制体积
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $OUTPUT_NAME ./cmd/main.go

if [ $? -eq 0 ]; then
    echo "编译成功！输出文件: $OUTPUT_NAME"
    chmod +x $OUTPUT_NAME
else
    echo "编译失败！"
    exit 1
fi
