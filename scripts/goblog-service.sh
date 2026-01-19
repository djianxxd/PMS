#!/bin/bash

# GoBlog SystemD 服务管理脚本
# 使用方法: ./goblog-service.sh {install|uninstall|enable|disable|start|stop|restart|status}

set -e

# 配置变量
SERVICE_NAME="goblog"
SERVICE_FILE="goblog.service"
SYSTEMD_DIR="/etc/systemd/system"
APP_DIR="/opt/goblog"
APP_USER="goblog"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 检查是否以root权限运行
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本需要root权限运行"
        exit 1
    fi
}

# 检查systemd是否可用
check_systemd() {
    if ! command -v systemctl >/dev/null 2>&1; then
        log_error "systemctl 命令不存在，此脚本适用于使用systemd的系统"
        exit 1
    fi
}

# 创建应用用户
create_user() {
    if ! id "$APP_USER" &>/dev/null; then
        log_info "创建应用用户: $APP_USER"
        useradd -r -s /bin/false -d "$APP_DIR" "$APP_USER"
    else
        log_info "应用用户 $APP_USER 已存在"
    fi
}

# 安装服务
install_service() {
    log_info "安装 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    create_user
    
    # 确保脚本目录存在
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    SERVICE_SOURCE="$SCRIPT_DIR/$SERVICE_FILE"
    
    if [[ ! -f "$SERVICE_SOURCE" ]]; then
        log_error "服务文件不存在: $SERVICE_SOURCE"
        exit 1
    fi
    
    # 复制服务文件
    cp "$SERVICE_SOURCE" "$SYSTEMD_DIR/"
    chmod 644 "$SYSTEMD_DIR/$SERVICE_FILE"
    
    # 重新加载systemd
    systemctl daemon-reload
    
    log_info "服务文件已安装到: $SYSTEMD_DIR/$SERVICE_FILE"
}

# 卸载服务
uninstall_service() {
    log_info "卸载 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    
    # 停止并禁用服务
    if systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
        systemctl stop "$SERVICE_NAME"
        log_info "服务已停止"
    fi
    
    if systemctl is-enabled --quiet "$SERVICE_NAME" 2>/dev/null; then
        systemctl disable "$SERVICE_NAME"
        log_info "服务已禁用"
    fi
    
    # 删除服务文件
    if [[ -f "$SYSTEMD_DIR/$SERVICE_FILE" ]]; then
        rm -f "$SYSTEMD_DIR/$SERVICE_FILE"
        systemctl daemon-reload
        log_info "服务文件已删除"
    fi
    
    # 删除用户 (可选)
    read -p "是否删除应用用户 $APP_USER? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        userdel -r "$APP_USER" 2>/dev/null || true
        log_info "用户 $APP_USER 已删除"
    fi
}

# 启用服务
enable_service() {
    log_info "启用 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    
    if systemctl enable "$SERVICE_NAME"; then
        log_info "服务已启用，将在系统启动时自动运行"
    else
        log_error "启用服务失败"
        exit 1
    fi
}

# 禁用服务
disable_service() {
    log_info "禁用 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    
    if systemctl disable "$SERVICE_NAME"; then
        log_info "服务已禁用，不会在系统启动时自动运行"
    else
        log_error "禁用服务失败"
        exit 1
    fi
}

# 启动服务
start_service() {
    log_info "启动 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    
    if systemctl start "$SERVICE_NAME"; then
        log_info "服务启动成功"
        
        # 检查状态
        sleep 2
        if systemctl is-active --quiet "$SERVICE_NAME"; then
            log_info "服务运行中"
            show_status
        else
            log_error "服务启动失败"
            systemctl status "$SERVICE_NAME"
            exit 1
        fi
    else
        log_error "启动服务失败"
        exit 1
    fi
}

# 停止服务
stop_service() {
    log_info "停止 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    
    if systemctl stop "$SERVICE_NAME"; then
        log_info "服务已停止"
    else
        log_error "停止服务失败"
        exit 1
    fi
}

# 重启服务
restart_service() {
    log_info "重启 $SERVICE_NAME 服务..."
    
    check_root
    check_systemd
    
    if systemctl restart "$SERVICE_NAME"; then
        log_info "服务重启成功"
        sleep 2
        show_status
    else
        log_error "重启服务失败"
        exit 1
    fi
}

# 显示服务状态
show_status() {
    echo "=== $SERVICE_NAME 服务状态 ==="
    systemctl status "$SERVICE_NAME" --no-pager -l
    
    echo ""
    echo "=== 服务详细信息 ==="
    
    # 检查服务是否启用
    if systemctl is-enabled --quiet "$SERVICE_NAME"; then
        echo -e "开机启动: ${GREEN}已启用${NC}"
    else
        echo -e "开机启动: ${RED}已禁用${NC}"
    fi
    
    # 检查服务是否运行
    if systemctl is-active --quiet "$SERVICE_NAME"; then
        echo -e "运行状态: ${GREEN}运行中${NC}"
        
        # 显示进程信息
        PID=$(systemctl show --property MainPID --value "$SERVICE_NAME")
        if [[ "$PID" != "0" ]]; then
            echo "进程ID: $PID"
            
            # 显示资源使用情况
            if command -v ps >/dev/null 2>&1; then
                ps -p "$PID" -o pid,pcpu,pmem,etime,cmd --no-headers 2>/dev/null || true
            fi
        fi
        
        # 检查端口监听
        if command -v netstat >/dev/null 2>&1; then
            if netstat -tln 2>/dev/null | grep -q ":8080 "; then
                echo -e "端口状态: ${GREEN}8080 (监听中)${NC}"
            else
                echo -e "端口状态: ${RED}8080 (未监听)${NC}"
            fi
        elif command -v ss >/dev/null 2>&1; then
            if ss -tln 2>/dev/null | grep -q ":8080 "; then
                echo -e "端口状态: ${GREEN}8080 (监听中)${NC}"
            else
                echo -e "端口状态: ${RED}8080 (未监听)${NC}"
            fi
        fi
    else
        echo -e "运行状态: ${RED}已停止${NC}"
    fi
    
    # 显示最近的日志
    echo ""
    echo "=== 最近日志 (最后5行) ==="
    journalctl -u "$SERVICE_NAME" --no-pager -n 5 --no-hostname -o cat 2>/dev/null || echo "无法获取日志"
}

# 显示帮助信息
show_help() {
    echo "GoBlog SystemD 服务管理脚本"
    echo ""
    echo "使用方法:"
    echo "  $0 {install|uninstall|enable|disable|start|stop|restart|status|help}"
    echo ""
    echo "命令说明:"
    echo "  install   - 安装systemd服务"
    echo "  uninstall - 卸载systemd服务"
    echo "  enable    - 启用服务开机启动"
    echo "  disable   - 禁用服务开机启动"
    echo "  start     - 启动服务"
    echo "  stop      - 停止服务"
    echo "  restart   - 重启服务"
    echo "  status    - 显示服务状态"
    echo "  help      - 显示此帮助信息"
    echo ""
    echo "服务配置:"
    echo "  服务名称: $SERVICE_NAME"
    echo "  服务文件: $SYSTEMD_DIR/$SERVICE_FILE"
    echo "  应用目录: $APP_DIR"
    echo "  运行用户: $APP_USER"
}

# 主函数
main() {
    case "${1:-help}" in
        install)
            install_service
            ;;
        uninstall)
            uninstall_service
            ;;
        enable)
            enable_service
            ;;
        disable)
            disable_service
            ;;
        start)
            start_service
            ;;
        stop)
            stop_service
            ;;
        restart)
            restart_service
            ;;
        status)
            show_status
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 脚本入口
main "$@"