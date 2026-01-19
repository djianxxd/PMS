#!/bin/bash

# GoBlog 构建脚本
# 支持多平台交叉编译

set -e

# 配置变量
APP_NAME="goblog"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')
LDFLAGS="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.GoVersion=$GO_VERSION"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 构建函数
build() {
    local os=$1
    local arch=$2
    local output_name=$3
    
    log_step "构建 $os/$arch..."
    
    # 设置输出文件名
    if [[ "$os" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi
    
    # 创建输出目录
    mkdir -p "build/$os-$arch"
    
    # 构建
    GOOS=$os GOARCH=$arch go build \
        -ldflags="$LDFLAGS" \
        -o "build/$os-$arch/$output_name" \
        .
    
    log_info "构建完成: build/$os-$arch/$output_name"
}

# 打包函数
package() {
    local os=$1
    local arch=$2
    local output_name=$3
    
    if [[ "$os" == "windows" ]]; then
        output_name="${output_name}.exe"
    fi
    
    log_step "打包 $os/$arch..."
    
    cd "build/$os-$arch"
    
    # 创建部署包
    mkdir -p package
    
    # 复制文件
    cp "$output_name" package/
    
    # 复制配置文件
    cp -r ../../config package/
    cp -r ../../scripts package/
    cp -r ../../templates package/ 2>/dev/null || true
    
    # 创建README
    cat > package/README.txt << EOF
GoBlog $VERSION - $os/$arch

安装说明:

1. 二进制文件:
   $output_name

2. 配置文件:
   config/ - 环境配置文件目录

3. 脚本文件:
   scripts/ - Linux启动和管理脚本

4. 模板文件:
   templates/ - Web界面模板文件

快速启动:

Linux/macOS:
  chmod +x $output_name
  ./$output_name

Windows:
  $output_name.exe

配置环境:
  cp config/development.env .env
  编辑 .env 文件修改配置

访问地址: http://localhost:8080

构建信息:
  版本: $VERSION
  构建时间: $BUILD_TIME
  Go版本: $GO_VERSION
EOF
    
    # 压缩打包
    if [[ "$os" == "windows" ]]; then
        zip -r "../../${APP_NAME}-${VERSION}-${os}-${arch}.zip" package/
    else
        tar -czf "../../${APP_NAME}-${VERSION}-${os}-${arch}.tar.gz" package/
    fi
    
    # 清理
    rm -rf package/
    
    cd - > /dev/null
    
    log_info "打包完成: ${APP_NAME}-${VERSION}-${os}-${arch}.tar.gz/.zip"
}

# 主函数
main() {
    log_info "开始构建 GoBlog..."
    log_info "版本: $VERSION"
    log_info "Go版本: $GO_VERSION"
    
    # 检查依赖
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go 未安装"
        exit 1
    fi
    
    # 下载依赖
    log_step "下载Go模块依赖..."
    go mod download
    go mod tidy
    
    # 运行测试
    if [[ "$1" != "--skip-tests" ]]; then
        log_step "运行测试..."
        go test -v ./...
    fi
    
    # 清理构建目录
    rm -rf build/
    
    # 构建目标平台
    declare -a platforms=(
        "linux:amd64:goblog"
        "linux:arm64:goblog"
        "darwin:amd64:goblog"
        "darwin:arm64:goblog"
        "windows:amd64:goblog"
        "windows:arm64:goblog"
    )
    
    for platform in "${platforms[@]}"; do
        IFS=':' read -r os arch output_name <<< "$platform"
        
        # 检查是否支持该平台
        if ! go env GOOS | grep -q "$os" || ! go env GOARCH | grep -q "$arch"; then
            log_warn "跳过不支持的 $os/$arch"
            continue
        fi
        
        build "$os" "$arch" "$output_name"
        
        # 如果指定了打包，则进行打包
        if [[ "$1" == "--package" || "$2" == "--package" ]]; then
            package "$os" "$arch" "$output_name"
        fi
    done
    
    # 显示构建结果
    echo ""
    log_info "构建完成！"
    echo ""
    echo "构建产物:"
    find build -name "$APP_NAME*" -type f | while read file; do
        size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
        echo "  $(basename "$(dirname "$file")")/$(basename "$file") (${size} bytes)"
    done
    
    if [[ "$1" == "--package" || "$2" == "--package" ]]; then
        echo ""
        echo "发布包:"
        find . -name "${APP_NAME}-*.tar.gz" -o -name "${APP_NAME}-*.zip" | while read file; do
            size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
            echo "  $(basename "$file") (${size} bytes)"
        done
    fi
    
    echo ""
    log_info "本地构建版本: build/$(go env GOOS)/$(go env GOARCH)/$APP_NAME"
}

# 显示帮助
show_help() {
    echo "GoBlog 构建脚本"
    echo ""
    echo "使用方法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --package        构建并打包发布版本"
    echo "  --skip-tests     跳过测试"
    echo "  -h, --help       显示帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                    # 本地构建"
    echo "  $0 --package          # 构建并打包所有平台"
    echo "  $0 --skip-tests       # 跳过测试构建"
}

# 解析命令行参数
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    *)
        main "$@"
        ;;
esac