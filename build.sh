#!/bin/bash

# 构建脚本 - 支持多平台构建

# 设置应用程序名称
APP_NAME="server-file"

# 创建dist目录存放构建产物
mkdir -p dist

echo "开始构建 $APP_NAME..."

# 构建AMD64架构版本 (Linux)
echo "构建 linux/amd64 版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/${APP_NAME}-linux-amd64 .

# 构建ARM64架构版本 (Linux)
echo "构建 linux/arm64 版本..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o dist/${APP_NAME}-linux-arm64 .

# 构建AMD64架构版本 (macOS)
echo "构建 darwin/amd64 版本..."
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o dist/${APP_NAME}-darwin-amd64 .

# 构建ARM64架构版本 (macOS/M1芯片)
echo "构建 darwin/arm64 版本..."
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o dist/${APP_NAME}-darwin-arm64 .

# 构建AMD64架构版本 (Windows)
echo "构建 windows/amd64 版本..."
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o dist/${APP_NAME}-windows-amd64.exe .

echo "所有平台构建完成!"
echo "构建产物位于 dist/ 目录下:"
ls -la dist/