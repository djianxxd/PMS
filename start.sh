#!/bin/bash

# GoBlog Linux启动脚本
# 自动下载依赖并启动服务

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查系统要求
check_requirements() {
    log_info "检查系统要求..."
    
    # 检查Go是否安装
    if ! command -v go &> /dev/null; then
        log_error "Go未安装！请先安装Go 1.21或更高版本"
        log_info "Ubuntu/Debian: sudo apt install golang-go"
        log_info "CentOS/RHEL: sudo yum install golang"
        log_info "或访问 https://golang.org/dl/ 下载"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_success "Go版本: $GO_VERSION"
    
    # 检查是否有足够的磁盘空间（至少100MB）
    AVAILABLE_SPACE=$(df . | awk 'NR==2 {print $4}')
    if [ "$AVAILABLE_SPACE" -lt 102400 ]; then
        log_warning "磁盘空间不足100MB，可能影响依赖下载"
    fi
}

# 配置Go环境（解决中国网络问题）
setup_go_env() {
    log_info "配置Go环境变量..."
    
    # 设置Go代理加速
    export GOPROXY=https://goproxy.cn,direct
    export GOSUMDB=sum.golang.google.cn
    export GO111MODULE=on
    
    # 添加到~/.bashrc（如果不存在）
    if ! grep -q "GOPROXY" ~/.bashrc; then
        echo "export GOPROXY=https://goproxy.cn,direct" >> ~/.bashrc
        echo "export GOSUMDB=sum.golang.google.cn" >> ~/.bashrc
        echo "export GO111MODULE=on" >> ~/.bashrc
        log_success "Go代理配置已添加到~/.bashrc"
    fi
    
    log_success "Go代理配置完成"
}

# 下载Go依赖
download_dependencies() {
    log_info "开始下载Go依赖包..."
    
    # 创建临时目录用于缓存
    mkdir -p tmp
    
    # 清理module缓存（可选，解决版本冲突）
    log_info "清理module缓存..."
    go clean -modcache 2>/dev/null || true
    
    # 下载依赖
    log_info "下载项目依赖..."
    if go mod download; then
        log_success "依赖下载成功"
    else
        log_error "依赖下载失败，尝试备用方案..."
        
        # 尝试直接下载关键依赖
        go get modernc.org/sqlite@v1.44.2
        go get github.com/google/uuid@v1.6.0
        go get github.com/dustin/go-humanize@v1.0.1
        
        log_info "重新尝试下载所有依赖..."
        go mod download
    fi
    
    # 验证依赖
    log_info "验证依赖完整性..."
    if go mod verify; then
        log_success "依赖验证通过"
    else
        log_warning "依赖验证失败，但继续启动..."
    fi
    
    # 整理go.mod和go.sum
    go mod tidy
}

# 预编译检查
pre_build_check() {
    log_info "执行预编译检查..."
    
    # 检查语法错误
    if go vet ./...; then
        log_success "代码检查通过"
    else
        log_error "代码检查失败，请修复错误后重试"
        exit 1
    fi
    
    # 格式化检查
    log_info "检查代码格式..."
    UNFORMATTED=$(gofmt -l . 2>/dev/null || true)
    if [ -n "$UNFORMATTED" ]; then
        log_warning "以下文件需要格式化："
        echo "$UNFORMATTED"
        log_info "执行自动格式化..."
        gofmt -w .
    fi
}

# 编译应用
build_application() {
    log_info "编译Go应用..."
    
    # 设置编译参数
    BUILD_FLAGS="-ldflags '-s -w' -trimpath"
    OUTPUT_BINARY="goblog"
    
    # 编译
    if go build $BUILD_FLAGS -o $OUTPUT_BINARY .; then
        log_success "编译成功: $OUTPUT_BINARY"
        
        # 检查可执行文件
        if [ -f "$OUTPUT_BINARY" ]; then
            chmod +x $OUTPUT_BINARY
            BINARY_SIZE=$(ls -lh $OUTPUT_BINARY | awk '{print $5}')
            log_success "可执行文件大小: $BINARY_SIZE"
        else
            log_error "编译失败：找不到可执行文件"
            exit 1
        fi
    else
        log_error "编译失败"
        exit 1
    fi
}

# 数据库初始化
init_database() {
    log_info "检查数据库文件..."
    
    # 检查db目录
    if [ ! -d "db" ]; then
        mkdir -p db
        log_info "创建db目录"
    fi
    
    # 数据库文件将在首次运行时自动创建
    log_success "数据库配置完成"
}

# 创建systemd服务（可选）
create_systemd_service() {
    if [ "$1" = "--service" ]; then
        log_info "创建systemd服务..."
        
        SERVICE_FILE="/etc/systemd/system/goblog.service"
        CURRENT_DIR=$(pwd)
        
        if [ "$EUID" -ne 0 ]; then
            log_warning "需要root权限创建systemd服务"
            log_info "请使用: sudo $0 --service"
            return
        fi
        
        cat > $SERVICE_FILE << EOF
[Unit]
Description=GoBlog Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$CURRENT_DIR
ExecStart=$CURRENT_DIR/goblog
Restart=always
RestartSec=5
Environment=GOPROXY=https://goproxy.cn,direct
Environment=GOSUMDB=sum.golang.google.cn

[Install]
WantedBy=multi-user.target
EOF
        
        systemctl daemon-reload
        systemctl enable goblog
        log_success "systemd服务创建完成"
        log_info "使用以下命令管理服务："
        log_info "  启动: sudo systemctl start goblog"
        log_info "  停止: sudo systemctl stop goblog"
        log_info "  状态: sudo systemctl status goblog"
        log_info "  日志: sudo journalctl -u goblog -f"
    fi
}

# 启动应用
start_application() {
    log_info "启动GoBlog应用..."
    
    # 检查端口是否被占用
    if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
        log_warning "端口8080已被占用，尝试终止现有进程..."
        pkill -f "goblog" 2>/dev/null || true
        sleep 2
    fi
    
    # 启动应用
    if [ -f "./goblog" ]; then
        log_success "启动服务器: http://localhost:8080"
        log_info "按 Ctrl+C 停止服务器"
        ./goblog
    else
        log_error "找不到可执行文件，请重新编译"
        exit 1
    fi
}

# 显示帮助信息
show_help() {
    echo "GoBlog Linux启动脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --service     创建systemd服务（需要root权限）"
    echo "  --build-only  仅编译，不启动"
    echo "  --clean       清理编译文件和缓存"
    echo "  --help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                # 完整启动流程"
    echo "  $0 --build-only   # 仅编译应用"
    echo "  sudo $0 --service # 创建系统服务"
}

# 清理函数
cleanup() {
    log_info "清理临时文件..."
    rm -rf tmp
    rm -f goblog
    go clean -cache 2>/dev/null || true
    log_success "清理完成"
}

# 主函数
main() {
    echo "=================================="
    echo "🚀 GoBlog Linux启动脚本"
    echo "=================================="
    
    case "$1" in
        --help)
            show_help
            exit 0
            ;;
        --clean)
            cleanup
            exit 0
            ;;
        --build-only)
            check_requirements
            setup_go_env
            download_dependencies
            pre_build_check
            build_application
            init_database
            log_success "构建完成！运行 ./goblog 启动应用"
            exit 0
            ;;
        --service)
            # 先构建，再创建服务
            check_requirements
            setup_go_env
            download_dependencies
            pre_build_check
            build_application
            init_database
            create_systemd_service --service
            exit 0
            ;;
        "")
            # 默认启动流程
            check_requirements
            setup_go_env
            download_dependencies
            pre_build_check
            build_application
            init_database
            start_application
            ;;
        *)
            log_error "未知选项: $1"
            show_help
            exit 1
            ;;
    esac
}

# 捕获中断信号
trap 'log_info "收到中断信号，正在退出..."; exit 130' INT TERM

# 执行主函数
main "$@"