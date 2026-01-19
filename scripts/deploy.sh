#!/bin/bash

# GoBlog 一键部署脚本
# 支持 CentOS, Ubuntu, Debian 等主流Linux发行版

set -e

# 配置变量
APP_NAME="goblog"
APP_USER="goblog"
APP_DIR="/opt/goblog"
APP_VERSION="latest"
PORT="8080"
INSTALL_MODE="binary"  # binary, source

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
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

log_success() {
    echo -e "${PURPLE}[SUCCESS]${NC} $1"
}

# 显示横幅
show_banner() {
    echo -e "${CYAN}"
    echo "╔══════════════════════════════════════════════════════════════╗"
    echo "║                        GoBlog 部署脚本                        ║"
    echo "║                   个人生活管理系统一键安装                    ║"
    echo "╚══════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# 检查是否以root权限运行
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本需要root权限运行，请使用 sudo $0"
        exit 1
    fi
}

# 检测操作系统
detect_os() {
    log_step "检测操作系统..."
    
    if [[ -f /etc/redhat-release ]]; then
        OS="centos"
        PKG_MANAGER="yum"
        log_info "检测到 CentOS/RHEL 系统"
    elif [[ -f /etc/debian_version ]]; then
        OS="debian"
        if command -v apt-get >/dev/null 2>&1; then
            PKG_MANAGER="apt-get"
        elif command -v apt >/dev/null 2>&1; then
            PKG_MANAGER="apt"
        fi
        log_info "检测到 Debian/Ubuntu 系统"
    else
        log_error "不支持的操作系统"
        exit 1
    fi
    
    # 获取系统版本
    if [[ "$OS" == "centos" ]]; then
        OS_VERSION=$(cat /etc/redhat-release | grep -oE '[0-9]+\.' | head -1 | sed 's/\.//')
    else
        OS_VERSION=$(cat /etc/debian_version | cut -d. -f1)
    fi
    
    log_info "操作系统: $OS, 版本: $OS_VERSION"
}

# 检查网络连接
check_network() {
    log_step "检查网络连接..."
    
    if ! ping -c 1 8.8.8.8 >/dev/null 2>&1; then
        log_error "网络连接失败，请检查网络设置"
        exit 1
    fi
    
    log_info "网络连接正常"
}

# 安装系统依赖
install_dependencies() {
    log_step "安装系统依赖..."
    
    case $OS in
        centos)
            if command -v dnf >/dev/null 2>&1; then
                PKG_MANAGER="dnf"
            fi
            $PKG_MANAGER update -y
            $PKG_MANAGER install -y wget curl tar gzip supervisor
            ;;
        debian)
            $PKG_MANAGER update
            $PKG_MANAGER install -y wget curl tar gzip supervisor systemd
            ;;
    esac
    
    log_info "系统依赖安装完成"
}

# 创建应用用户
create_user() {
    log_step "创建应用用户..."
    
    if id "$APP_USER" &>/dev/null; then
        log_warn "用户 $APP_USER 已存在"
    else
        useradd -r -s /bin/false -d "$APP_DIR" -m "$APP_USER"
        log_info "用户 $APP_USER 创建成功"
    fi
}

# 创建应用目录
create_directories() {
    log_step "创建应用目录..."
    
    mkdir -p "$APP_DIR"
    mkdir -p "$APP_DIR/data"
    mkdir -p "$APP_DIR/data/backups"
    mkdir -p "$APP_DIR/scripts"
    mkdir -p "/var/log/$APP_NAME"
    
    # 设置权限
    chown -R "$APP_USER:$APP_USER" "$APP_DIR"
    chown -R "$APP_USER:$APP_USER" "/var/log/$APP_NAME"
    
    log_info "目录创建完成"
}

# 安装Go环境
install_go() {
    log_step "检查Go环境..."
    
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | head -1)
        log_info "Go已安装: $GO_VERSION"
    else
        log_info "安装Go环境..."
        GO_VERSION="1.21.5"
        
        cd /tmp
        wget -q "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz"
        tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
        
        # 添加到系统PATH
        echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
        export PATH=$PATH:/usr/local/go/bin
        
        # 创建符号链接
        ln -sf /usr/local/go/bin/go /usr/bin/go
        ln -sf /usr/local/go/bin/gofmt /usr/bin/gofmt
        
        rm -f "go${GO_VERSION}.linux-amd64.tar.gz"
        
        log_info "Go ${GO_VERSION} 安装完成"
    fi
}

# 下载应用二进制文件
download_binary() {
    log_step "下载应用二进制文件..."
    
    cd /tmp
    
    # 这里假设你已经构建了二进制文件，实际使用时需要替换为真实的下载URL
    # wget -q "https://github.com/your-repo/goblog/releases/download/v1.0.0/goblog-linux-amd64.tar.gz"
    # tar -xzf "goblog-linux-amd64.tar.gz"
    
    log_warn "请确保已在本地构建了 goblog 二进制文件"
    log_warn "将构建好的二进制文件复制到 $APP_DIR/goblog"
    
    # 如果二进制文件不存在，尝试从当前目录复制
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "$SCRIPT_DIR/../goblog" ]]; then
        cp "$SCRIPT_DIR/../goblog" "$APP_DIR/"
        chmod +x "$APP_DIR/goblog"
        log_info "二进制文件复制完成"
    elif [[ -f "$SCRIPT_DIR/../goblog.exe" ]]; then
        log_warn "检测到Windows可执行文件，需要在Linux环境下重新构建"
    else
        log_error "未找到应用二进制文件"
        log_info "请手动构建应用或将二进制文件放到 $APP_DIR/goblog"
        return 1
    fi
}

# 从源码编译
build_from_source() {
    log_step "从源码编译应用..."
    
    cd /tmp
    
    # 克隆源码 (这里需要替换为真实的仓库地址)
    # git clone https://github.com/your-repo/goblog.git
    # cd goblog
    
    # 从当前目录复制源码
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SOURCE_DIR="$SCRIPT_DIR/.."
    
    if [[ -d "$SOURCE_DIR" && -f "$SOURCE_DIR/main.go" ]]; then
        cp -r "$SOURCE_DIR" /tmp/goblog
        cd /tmp/goblog
        
        # 下载依赖
        go mod tidy
        
        # 构建
        go build -ldflags="-s -w" -o goblog .
        
        # 复制到应用目录
        cp goblog "$APP_DIR/"
        chmod +x "$APP_DIR/goblog"
        
        # 清理
        rm -rf /tmp/goblog
        
        log_info "源码编译完成"
    else
        log_error "未找到源码目录"
        return 1
    fi
}

# 安装应用文件
install_app() {
    log_step "安装应用文件..."
    
    case $INSTALL_MODE in
        binary)
            download_binary
            ;;
        source)
            build_from_source
            ;;
        *)
            log_error "未知的安装模式: $INSTALL_MODE"
            exit 1
            ;;
    esac
    
    # 复制脚本文件
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    if [[ -f "$SCRIPT_DIR/goblog.sh" ]]; then
        cp "$SCRIPT_DIR/goblog.sh" "$APP_DIR/scripts/"
        chmod +x "$APP_DIR/scripts/goblog.sh"
    fi
    
    if [[ -f "$SCRIPT_DIR/goblog-service.sh" ]]; then
        cp "$SCRIPT_DIR/goblog-service.sh" "$APP_DIR/scripts/"
        chmod +x "$APP_DIR/scripts/goblog-service.sh"
    fi
    
    # 创建环境配置文件
    cat > "$APP_DIR/.env" << EOF
# GoBlog 环境配置文件
APP_NAME=$APP_NAME
APP_ENV=production
APP_PORT=$PORT
APP_HOST=0.0.0.0

# 数据库配置
DB_PATH=$APP_DIR/data/app.db

# 日志配置
LOG_PATH=/var/log/$APP_NAME/goblog.log
LOG_LEVEL=info

# 备份配置
BACKUP_DIR=$APP_DIR/data/backups
AUTO_BACKUP=true
BACKUP_INTERVAL=24h
EOF
    
    chown -R "$APP_USER:$APP_USER" "$APP_DIR"
    log_info "应用文件安装完成"
}

# 安装systemd服务
install_service() {
    log_step "安装systemd服务..."
    
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    
    if [[ -f "$SCRIPT_DIR/goblog.service" ]]; then
        cp "$SCRIPT_DIR/goblog.service" "/etc/systemd/system/"
        chmod 644 "/etc/systemd/system/goblog.service"
        
        systemctl daemon-reload
        systemctl enable goblog
        
        log_info "systemd服务安装完成"
    else
        log_warn "服务文件不存在，跳过服务安装"
    fi
}

# 配置防火墙
configure_firewall() {
    log_step "配置防火墙..."
    
    if command -v firewall-cmd >/dev/null 2>&1; then
        # CentOS/RHEL with firewalld
        if systemctl is-active --quiet firewalld; then
            firewall-cmd --permanent --add-port="$PORT/tcp"
            firewall-cmd --reload
            log_info "防火墙配置完成 (firewalld)"
        fi
    elif command -v ufw >/dev/null 2>&1; then
        # Ubuntu/Debian with ufw
        if ufw status | grep -q "Status: active"; then
            ufw allow "$PORT/tcp"
            log_info "防火墙配置完成 (ufw)"
        fi
    else
        log_warn "未检测到防火墙管理工具，请手动开放端口 $PORT"
    fi
}

# 启动应用
start_app() {
    log_step "启动应用..."
    
    if systemctl is-active --quiet goblog; then
        systemctl stop goblog
    fi
    
    systemctl start goblog
    
    # 等待应用启动
    sleep 3
    
    if systemctl is-active --quiet goblog; then
        log_success "应用启动成功！"
        log_info "访问地址: http://localhost:$PORT"
        log_info "服务状态: systemctl status goblog"
        log_info "查看日志: journalctl -u goblog -f"
    else
        log_error "应用启动失败"
        systemctl status goblog
        exit 1
    fi
}

# 显示安装信息
show_install_info() {
    echo ""
    echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║                        安装完成！                          ║${NC}"
    echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${GREEN}应用信息:${NC}"
    echo "  应用名称: $APP_NAME"
    echo "  应用目录: $APP_DIR"
    echo "  运行用户: $APP_USER"
    echo "  访问地址: http://localhost:$PORT"
    echo ""
    echo -e "${GREEN}管理命令:${NC}"
    echo "  启动服务: systemctl start goblog"
    echo "  停止服务: systemctl stop goblog"
    echo "  重启服务: systemctl restart goblog"
    echo "  查看状态: systemctl status goblog"
    echo "  查看日志: journalctl -u goblog -f"
    echo ""
    echo -e "${GREEN}脚本管理:${NC}"
    echo "  手动脚本: $APP_DIR/scripts/goblog.sh"
    echo "  服务脚本: $APP_DIR/scripts/goblog-service.sh"
    echo ""
    echo -e "${GREEN}数据目录:${NC}"
    echo "  数据库: $APP_DIR/data/app.db"
    echo "  备份目录: $APP_DIR/data/backups"
    echo "  日志目录: /var/log/$APP_NAME"
    echo ""
}

# 清理函数
cleanup() {
    log_info "清理临时文件..."
    rm -rf /tmp/goblog* 2>/dev/null || true
}

# 主函数
main() {
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --mode=*)
                INSTALL_MODE="${1#*=}"
                shift
                ;;
            --port=*)
                PORT="${1#*=}"
                shift
                ;;
            --user=*)
                APP_USER="${1#*=}"
                shift
                ;;
            --dir=*)
                APP_DIR="${1#*=}"
                shift
                ;;
            -h|--help)
                echo "GoBlog 一键部署脚本"
                echo ""
                echo "使用方法: $0 [选项]"
                echo ""
                echo "选项:"
                echo "  --mode=binary|source   安装模式 (默认: binary)"
                echo "  --port=PORT           监听端口 (默认: 8080)"
                echo "  --user=USER           运行用户 (默认: goblog)"
                echo "  --dir=DIR             安装目录 (默认: /opt/goblog)"
                echo "  -h, --help            显示帮助信息"
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                exit 1
                ;;
        esac
    done
    
    # 设置错误处理
    trap cleanup EXIT
    
    show_banner
    
    log_step "开始部署 GoBlog..."
    
    # 检查环境
    check_root
    detect_os
    check_network
    
    # 安装依赖
    install_dependencies
    create_user
    create_directories
    
    # 根据安装模式安装Go
    if [[ "$INSTALL_MODE" == "source" ]]; then
        install_go
    fi
    
    # 安装应用
    install_app
    install_service
    configure_firewall
    
    # 启动应用
    start_app
    
    # 显示安装信息
    show_install_info
    
    log_success "GoBlog 部署完成！"
}

# 脚本入口
main "$@"