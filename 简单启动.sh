#!/bin/bash

# 自律人生 - 简单启动脚本

echo "自律人生 - 个人生活管理系统"
echo "================================"
echo

# 检查可执行文件
if [ ! -f "自律人生-linux" ]; then
    echo "错误：找不到'自律人生-linux'文件！"
    echo "请确保程序文件在当前目录中。"
    exit 1
fi

# 添加执行权限
if [ ! -x "自律人生-linux" ]; then
    echo "正在添加执行权限..."
    chmod +x "自律人生-linux"
fi

# 创建数据目录
if [ ! -d "data" ]; then
    mkdir -p "data"
fi

echo "正在启动服务器..."
echo

# 启动程序
./自律人生-linux

# 检查退出状态
if [ $? -ne 0 ]; then
    echo ""
    echo "程序异常退出"
    echo "请检查："
    echo "1. 端口8080是否被占用"
    echo "2. 是否有足够的权限"
    echo "3. data目录是否可写"
fi