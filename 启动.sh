#!/bin/bash

# 自律人生 - Linux启动脚本
# 作者: 自律人生开发团队
# 版本: 1.0

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 打印彩色消息
print_message() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# 打印分隔线
print_separator() {
    echo -e "${BLUE}========================================${NC}"
}

# 检查程序是否已运行
check_running() {
    if pgrep -f "自律人生-linux" > /dev/null; then
        print_message $YELLOW "检测到程序已在运行中..."
        print_message $CYAN "进程信息:"
        ps aux | grep "自律人生-linux" | grep -v grep
        echo ""
        print_message $GREEN "请访问: http://localhost:8080"
        echo ""
        
        read -p "是否要重新启动程序？(y/n): " -n 1 -r
        echo ""
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_message $BLUE "程序继续运行中..."
            exit 0
        fi
        
        print_message $YELLOW "正在停止现有进程..."
        pkill -f "自律人生-linux"
        sleep 2
    fi
}

# 检查端口是否被占用
check_port() {
    if command -v netstat &> /dev/null; then
        if netstat -tuln | grep -q ":8080 "; then
            print_message $YELLOW "警告：端口8080可能被其他程序占用！"
            print_message $CYAN "占用端口的进程信息:"
            netstat -tulpn | grep ":8080"
            echo ""
            
            read -p "是否继续启动？(y/n): " -n 1 -r
            echo ""
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                print_message $BLUE "已取消启动"
                exit 1
            fi
        fi
    elif command -v ss &> /dev/null; then
        if ss -tuln | grep -q ":8080 "; then
            print_message $YELLOW "警告：端口8080可能被其他程序占用！"
            ss -tulpn | grep ":8080"
            echo ""
            
            read -p "是否继续启动？(y/n): " -n 1 -r
            echo ""
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                print_message $BLUE "已取消启动"
                exit 1
            fi
        fi
    fi
}

# 检查依赖
check_dependencies() {
    # 检查可执行文件
    if [ ! -f "自律人生-linux" ]; then
        print_message $RED "错误：找不到'自律人生-linux'文件！"
        print_message $YELLOW "请确保程序文件在当前目录中。"
        exit 1
    fi
    
    # 检查执行权限
    if [ ! -x "自律人生-linux" ]; then
        print_message $YELLOW "正在添加执行权限..."
        chmod +x "自律人生-linux"
    fi
    
    # 检查数据目录
    if [ ! -d "data" ]; then
        print_message $CYAN "正在创建数据目录..."
        mkdir -p "data"
    fi
}

# 显示启动信息
show_banner() {
    clear
    print_separator
    print_message $PURPLE "        自律人生 - 个人生活管理系统"
    print_separator
    print_message $CYAN "版本: 1.0"
    print_message $CYAN "平台: Linux"
    print_message $CYAN "端口: 8080"
    print_separator
    echo ""
}

# 启动程序
start_program() {
    print_message $GREEN "正在启动服务器..."
    echo ""
    
    # 启动程序并捕获输出
    ./自律人生-linux 2>&1
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        echo ""
        print_separator
        print_message $RED "程序异常退出，错误代码: $exit_code"
        print_separator
        echo ""
        print_message $YELLOW "可能的解决方案："
        print_message $CYAN "1. 检查端口8080是否被其他程序占用"
        print_message $CYAN "2. 确保有足够的系统权限"
        print_message $CYAN "3. 检查防火墙设置"
        print_message $CYAN "4. 确保data目录有写入权限"
        print_message $CYAN "5. 检查系统资源使用情况"
        echo ""
        print_message $CYAN "查看详细错误信息:"
        print_message $YELLOW "dmesg | tail -20  # 查看系统日志"
        print_message $YELLOW "journalctl -xe     # 查看systemd日志"
        echo ""
        
        read -p "按任意键继续..." -n 1 -r
        echo ""
    fi
}

# 主函数
main() {
    # 显示启动信息
    show_banner
    
    # 系统检查
    check_dependencies
    check_running
    check_port
    
    # 启动程序
    start_program
}

# 捕获中断信号
trap 'print_message $YELLOW "正在停止程序..."; pkill -f "自律人生-linux"; exit 0' INT TERM

# 运行主函数
main "$@"