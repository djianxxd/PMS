# GoBlog Linux启动指南

## 🚀 快速启动

### 1. 一键启动（推荐）
```bash
chmod +x start.sh
./start.sh
```

### 2. 快速启动（适用于已配置环境）
```bash
chmod +x quick-start.sh
./quick-start.sh
```

### 3. 离线环境启动
```bash
# 先下载依赖
chmod +x download-deps.sh
./download-deps.sh

# 再启动
./quick-start.sh
```

## 📋 脚本说明

### start.sh - 完整启动脚本
- ✅ 检查系统要求
- ✅ 配置Go环境变量
- ✅ 下载所有依赖
- ✅ 代码检查和格式化
- ✅ 编译应用
- ✅ 启动服务

**选项：**
- `--service` - 创建systemd服务（需要root权限）
- `--build-only` - 仅编译，不启动
- `--clean` - 清理编译文件和缓存
- `--help` - 显示帮助信息

### quick-start.sh - 快速启动脚本
- ⚡ 快速依赖检查
- ⚡ 直接编译启动
- ⚡ 适用于重复启动

### download-deps.sh - 离线依赖下载
- 📦 提前下载所有依赖
- 📦 解决网络问题
- 📦 创建vendor目录

## 🛠️ 系统要求

- Go 1.21+ 
- Linux系统
- 至少100MB磁盘空间

## 🔧 环境配置

脚本会自动配置以下环境变量：
```bash
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=sum.golang.google.cn
export GO111MODULE=on
```

## 📦 依赖包

项目使用的主要依赖：
- `modernc.org/sqlite v1.44.2` - SQLite数据库
- `github.com/google/uuid v1.6.0` - UUID生成
- `github.com/dustin/go-humanize v1.0.1` - 字符串格式化

## 🌐 网络问题解决方案

如果遇到网络问题：

1. **使用代理脚本**
   ```bash
   ./download-deps.sh  # 先下载依赖
   ./quick-start.sh    # 再启动
   ```

2. **手动配置代理**
   ```bash
   export GOPROXY=https://goproxy.cn,direct
   go mod download
   ```

3. **使用systemd服务**
   ```bash
   sudo ./start.sh --service
   sudo systemctl start goblog
   ```

## 🚨 故障排除

### Go未安装
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install golang-go

# CentOS/RHEL
sudo yum install golang

# 或从官网下载
# https://golang.org/dl/
```

### 端口被占用
```bash
# 查看占用进程
lsof -i :8080

# 终止进程
pkill -f goblog
```

### 权限问题
```bash
chmod +x *.sh
```

### 清理缓存
```bash
./start.sh --clean
```

## 🎯 使用示例

### 首次启动
```bash
chmod +x start.sh
./start.sh
```

### 开发时快速重启
```bash
./quick-start.sh
```

### 生产环境部署
```bash
sudo ./start.sh --service
sudo systemctl start goblog
```

### 查看服务状态
```bash
sudo systemctl status goblog
sudo journalctl -u goblog -f
```

## 📞 访问地址

启动后访问：http://localhost:8080

## 🔄 更新依赖

```bash
go mod tidy
go mod download
```

---

**提示：** 如果网络环境较差，建议先运行 `./download-deps.sh` 下载所有依赖，再使用 `./quick-start.sh` 启动。