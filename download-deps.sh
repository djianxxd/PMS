#!/bin/bash

# GoBlog 离线依赖下载脚本
# 提前下载所有依赖，解决网络问题

set -e

echo "📦 GoBlog 依赖下载脚本"
echo "适用于网络环境较差的情况"

# 配置代理
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
export GO111MODULE=on

# 创建vendor目录
echo "📁 创建vendor目录..."
mkdir -p vendor

# 下载所有依赖到vendor
echo "⬇️  下载所有依赖..."
go mod vendor

# 下载特定版本依赖
echo "🎯 下载关键依赖..."

# 主要依赖
go get -d modernc.org/sqlite@v1.44.2
go get -d github.com/google/uuid@v1.6.0
go get -d github.com/dustin/go-humanize@v1.0.1
go get -d github.com/mattn/go-isatty@v0.0.20
go get -d github.com/ncruces/go-strftime@v1.0.0

# 间接依赖
go get -d github.com/remyoudompheng/bigfft@v0.0.0-20230129092748-24d4a6f8daec
go get -d golang.org/x/exp@v0.0.0-20251023183803-a4bb9ffd2546
go get -d golang.org/x/sys@v0.37.0
go get -d modernc.org/libc@v1.67.6
go get -d modernc.org/mathutil@v1.7.1
go get -d modernc.org/memory@v1.11.0

# 验证依赖
echo "✅ 验证依赖完整性..."
go mod verify

# 整理
go mod tidy

echo "🎉 所有依赖下载完成！"
echo "现在可以运行 ./start.sh 或 ./quick-start.sh 启动应用"