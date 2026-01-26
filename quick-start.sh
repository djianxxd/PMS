#!/bin/bash

# GoBlog 快速启动脚本
# 适用于已配置好环境的快速启动

set -e

echo "🚀 GoBlog 快速启动"

# 设置Go代理（解决网络问题）
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
export GO111MODULE=on

# 快速依赖检查和下载
if [ ! -f "go.sum" ] || [ ! -d "vendor" ]; then
    echo "📦 下载依赖..."
    go mod download
    go mod tidy
fi

# 编译
echo "🔨 编译应用..."
go build -o goblog .

# 启动
echo "✅ 启动服务器: http://localhost:8080"
echo "按 Ctrl+C 停止"
./goblog