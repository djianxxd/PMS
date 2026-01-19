#!/bin/bash

# GoBlog 启动管理脚本
# 使用方法: ./goblog.sh {start|stop|restart|status|logs}

set -e

# 配置变量
APP_NAME="goblog"
APP_USER="goblog"
APP_DIR="/opt/goblog"
APP_EXEC="$APP_DIR/goblog"
PID_FILE="/var/run/$APP_NAME.pid"
LOG_FILE="/var/log/$APP_NAME.log"
CONFIG_FILE="$APP_DIR/.env"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 检查是否以root权限运行
check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本需要root权限运行"
        exit 1
    fi
}

# 检查应用是否存在
check_app() {
    if [[ ! -f "$APP_EXEC" ]]; then
        log_error "应用文件不存在: $APP_EXEC"
        exit 1
    fi
}

# 检查PID文件
check_pid() {
    if [[ -f "$PID_FILE" ]]; then
        PID=$(cat "$PID_FILE")
        if ps -p "$PID" > /dev/null 2>&1; then
            return 0
        else
            rm -f "$PID_FILE"
            return 1
        fi
    else
        return 1
    fi
}

# 启动应用
start_app() {
    log_info "启动 $APP_NAME..."
    
    if check_pid; then
        log_warn "$APP_NAME 已在运行 (PID: $(cat $PID_FILE))"
        return 0
    fi
    
    # 创建必要的目录
    mkdir -p "$(dirname "$LOG_FILE")"
    mkdir -p "$APP_DIR/data"
    mkdir -p "$APP_DIR/data/backups"
    
    # 设置权限
    chown -R "$APP_USER:$APP_USER" "$APP_DIR" 2>/dev/null || true
    chown -R "$APP_USER:$APP_USER" "$(dirname "$LOG_FILE")" 2>/dev/null || true
    
    # 启动应用
    if command -v "$APP_USER" >/dev/null 2>&1; then
        # 如果用户存在，以该用户身份运行
        cd "$APP_DIR"
        nohup sudo -u "$APP_USER" "$APP_EXEC" >> "$LOG_FILE" 2>&1 &
    else
        # 如果用户不存在，以当前用户身份运行
        cd "$APP_DIR"
        nohup "$APP_EXEC" >> "$LOG_FILE" 2>&1 &
    fi
    
    PID=$!
    echo "$PID" > "$PID_FILE"
    
    # 等待应用启动
    sleep 2
    
    if check_pid; then
        log_info "$APP_NAME 启动成功 (PID: $PID)"
        log_info "日志文件: $LOG_FILE"
        log_info "访问地址: http://localhost:8080"
    else
        log_error "$APP_NAME 启动失败"
        rm -f "$PID_FILE"
        exit 1
    fi
}

# 停止应用
stop_app() {
    log_info "停止 $APP_NAME..."
    
    if ! check_pid; then
        log_warn "$APP_NAME 未在运行"
        return 0
    fi
    
    PID=$(cat "$PID_FILE")
    kill "$PID"
    
    # 等待进程结束
    for i in {1..10}; do
        if ! ps -p "$PID" > /dev/null 2>&1; then
            break
        fi
        sleep 1
    done
    
    # 如果进程仍在运行，强制杀死
    if ps -p "$PID" > /dev/null 2>&1; then
        log_warn "强制停止 $APP_NAME..."
        kill -9 "$PID"
        sleep 1
    fi
    
    rm -f "$PID_FILE"
    log_info "$APP_NAME 已停止"
}

# 重启应用
restart_app() {
    log_info "重启 $APP_NAME..."
    stop_app
    sleep 2
    start_app
}

# 检查应用状态
status_app() {
    echo "=== $APP_NAME 状态 ==="
    
    if check_pid; then
        PID=$(cat "$PID_FILE")
        echo -e "状态: ${GREEN}运行中${NC}"
        echo "PID: $PID"
        
        # 显示进程信息
        ps -p "$PID" -o pid,ppid,cmd,etime,pcpu,pmem --no-headers
        
        # 检查端口
        if command -v netstat >/dev/null 2>&1; then
            PORT_STATUS=$(netstat -tlnp 2>/dev/null | grep ":8080.*$PID" || echo "")
            if [[ -n "$PORT_STATUS" ]]; then
                echo -e "端口: ${GREEN}8080 (监听中)${NC}"
            else
                echo -e "端口: ${RED}8080 (未监听)${NC}"
            fi
        elif command -v ss >/dev/null 2>&1; then
            PORT_STATUS=$(ss -tlnp 2>/dev/null | grep ":8080.*$PID" || echo "")
            if [[ -n "$PORT_STATUS" ]]; then
                echo -e "端口: ${GREEN}8080 (监听中)${NC}"
            else
                echo -e "端口: ${RED}8080 (未监听)${NC}"
            fi
        fi
        
        # 检查日志文件
        if [[ -f "$LOG_FILE" ]]; then
            LOG_SIZE=$(stat -c%s "$LOG_FILE" 2>/dev/null || echo "0")
            echo "日志大小: ${LOG_SIZE} 字节"
            echo "日志文件: $LOG_FILE"
        fi
        
    else
        echo -e "状态: ${RED}未运行${NC}"
        
        if [[ -f "$PID_FILE" ]]; then
            echo -e "警告: ${YELLOW}PID文件存在但进程不存在${NC}"
            echo "PID文件: $PID_FILE"
        fi
    fi
    
    echo ""
    echo "=== 配置信息 ==="
    echo "应用目录: $APP_DIR"
    echo "执行文件: $APP_EXEC"
    echo "配置文件: $CONFIG_FILE"
    echo "日志文件: $LOG_FILE"
    
    # 检查数据库文件
    if [[ -f "$APP_DIR/data/app.db" ]]; then
        DB_SIZE=$(stat -c%s "$APP_DIR/data/app.db" 2>/dev/null || echo "0")
        echo "数据库: $APP_DIR/data/app.db (${DB_SIZE} 字节)"
    else
        echo -e "数据库: ${RED}不存在${NC}"
    fi
}

# 显示日志
show_logs() {
    if [[ -f "$LOG_FILE" ]]; then
        echo "=== $APP_NAME 日志 (最后50行) ==="
        tail -50 "$LOG_FILE"
    else
        log_error "日志文件不存在: $LOG_FILE"
    fi
}

# 显示帮助信息
show_help() {
    echo "GoBlog 启动管理脚本"
    echo ""
    echo "使用方法:"
    echo "  $0 {start|stop|restart|status|logs|help}"
    echo ""
    echo "命令说明:"
    echo "  start   - 启动应用"
    echo "  stop    - 停止应用"
    echo "  restart - 重启应用"
    echo "  status  - 查看应用状态"
    echo "  logs    - 查看应用日志"
    echo "  help    - 显示此帮助信息"
    echo ""
    echo "配置信息:"
    echo "  应用目录: $APP_DIR"
    echo "  执行文件: $APP_EXEC"
    echo "  PID文件: $PID_FILE"
    echo "  日志文件: $LOG_FILE"
    echo "  运行用户: $APP_USER"
}

# 主函数
main() {
    case "${1:-help}" in
        start)
            check_root
            check_app
            start_app
            ;;
        stop)
            check_root
            stop_app
            ;;
        restart)
            check_root
            check_app
            restart_app
            ;;
        status)
            status_app
            ;;
        logs)
            show_logs
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