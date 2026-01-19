# GoBlog Linux 部署文档

本文档介绍如何在 Linux 系统上部署和运行 GoBlog 应用。

## 🚀 快速部署

### 方法一：一键部署脚本（推荐）

```bash
# 下载部署脚本
wget https://raw.githubusercontent.com/your-repo/goblog/main/scripts/deploy.sh

# 或者克隆仓库
git clone https://github.com/your-repo/goblog.git
cd goblog/scripts

# 运行部署脚本
sudo ./deploy.sh
```

### 方法二：手动部署

1. **下载应用**
   ```bash
   # 下载预编译版本
   wget https://github.com/your-repo/goblog/releases/download/v1.0.0/goblog-1.0.0-linux-amd64.tar.gz
   tar -xzf goblog-1.0.0-linux-amd64.tar.gz
   ```

2. **创建应用目录**
   ```bash
   sudo mkdir -p /opt/goblog
   sudo cp goblog /opt/goblog/
   sudo cp -r config templates /opt/goblog/
   ```

3. **创建应用用户**
   ```bash
   sudo useradd -r -s /bin/false -d /opt/goblog goblog
   sudo chown -R goblog:goblog /opt/goblog
   ```

4. **配置环境**
   ```bash
   sudo cp config/production.env /opt/goblog/.env
   sudo chown goblog:goblog /opt/goblog/.env
   ```

## 📁 目录结构

```
/opt/goblog/
├── goblog                    # 主程序
├── .env                      # 环境配置
├── data/                     # 数据目录
│   ├── app.db               # 数据库文件
│   └── backups/             # 备份目录
├── config/                   # 配置文件
│   ├── production.env       # 生产环境配置
│   ├── development.env      # 开发环境配置
│   └── .env.example        # 配置示例
├── scripts/                  # 管理脚本
│   ├── goblog.sh            # 手动启动脚本
│   ├── goblog-service.sh    # SystemD服务脚本
│   └── deploy.sh            # 部署脚本
└── templates/                # Web模板
    ├── layout.html
    ├── dashboard.html
    ├── finance.html
    ├── habits.html
    └── todos.html
```

## ⚙️ 服务管理

### SystemD 服务（推荐）

```bash
# 启动服务
sudo systemctl start goblog

# 停止服务
sudo systemctl stop goblog

# 重启服务
sudo systemctl restart goblog

# 查看状态
sudo systemctl status goblog

# 设置开机启动
sudo systemctl enable goblog

# 查看日志
sudo journalctl -u goblog -f
```

### 手动启动脚本

```bash
# 使用管理脚本
cd /opt/goblog/scripts

# 启动应用
sudo ./goblog.sh start

# 停止应用
sudo ./goblog.sh stop

# 重启应用
sudo ./goblog.sh restart

# 查看状态
sudo ./goblog.sh status

# 查看日志
sudo ./goblog.sh logs
```

## 🔧 配置说明

### 环境变量配置

主要配置文件：`/opt/goblog/.env`

```bash
# 应用基础配置
APP_NAME=goblog
APP_ENV=production
APP_PORT=8080
APP_HOST=0.0.0.0

# 数据库配置
DB_PATH=/opt/goblog/data/app.db
DB_BACKUP_INTERVAL=24h

# 日志配置
LOG_LEVEL=info
LOG_PATH=/var/log/goblog/goblog.log
LOG_MAX_SIZE=100MB

# 备份配置
BACKUP_DIR=/opt/goblog/data/backups
AUTO_BACKUP=true
BACKUP_RETENTION_DAYS=30
```

### 防火墙配置

```bash
# CentOS/RHEL (firewalld)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# Ubuntu/Debian (ufw)
sudo ufw allow 8080/tcp
```

## 📊 监控和日志

### 查看应用状态

```bash
# SystemD 状态
sudo systemctl status goblog

# 进程信息
ps aux | grep goblog

# 端口监听
sudo netstat -tlnp | grep :8080
# 或
sudo ss -tlnp | grep :8080
```

### 日志管理

```bash
# SystemD 日志
sudo journalctl -u goblog -f          # 实时日志
sudo journalctl -u goblog -n 100     # 最近100行
sudo journalctl -u goblog --since "2024-01-01"  # 指定日期

# 应用日志
tail -f /var/log/goblog/goblog.log
```

### 健康检查

```bash
# 检查应用是否正常响应
curl -f http://localhost:8080/health || echo "应用异常"
```

## 🔄 更新应用

### 方法一：使用部署脚本更新

```bash
cd /opt/goblog/scripts
sudo ./deploy.sh --mode=binary
```

### 方法二：手动更新

```bash
# 1. 停止服务
sudo systemctl stop goblog

# 2. 备份当前版本
sudo cp /opt/goblog/goblog /opt/goblog/goblog.backup

# 3. 下载新版本
wget -O /tmp/goblog-new https://github.com/your-repo/goblog/releases/latest/download/goblog-linux-amd64

# 4. 替换程序
sudo mv /tmp/goblog-new /opt/goblog/goblog
sudo chmod +x /opt/goblog/goblog
sudo chown goblog:goblog /opt/goblog/goblog

# 5. 启动服务
sudo systemctl start goblog

# 6. 验证更新
sudo systemctl status goblog
```

## 🛡️ 安全配置

### 文件权限

```bash
# 设置正确的文件权限
sudo chmod +x /opt/goblog/goblog
sudo chmod 644 /opt/goblog/.env
sudo chmod -R 755 /opt/goblog/scripts/
sudo chown -R goblog:goblog /opt/goblog/
```

### SSL/TLS 配置

1. **使用反向代理（Nginx）**
   ```nginx
   server {
       listen 443 ssl http2;
       server_name your-domain.com;
       
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
       
       location / {
           proxy_pass http://127.0.0.1:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

2. **直接 HTTPS 配置**
   ```bash
   # 编辑 .env 文件
   ENABLE_HTTPS=true
   CERT_FILE=/path/to/cert.pem
   KEY_FILE=/path/to/key.pem
   ```

## 🚨 故障排除

### 常见问题

1. **应用无法启动**
   ```bash
   # 检查日志
   sudo journalctl -u goblog -n 50
   
   # 检查配置
   sudo -u goblog /opt/goblog/goblog --config-check
   ```

2. **数据库权限问题**
   ```bash
   sudo chown -R goblog:goblog /opt/goblog/data/
   sudo chmod 755 /opt/goblog/data/
   ```

3. **端口被占用**
   ```bash
   # 查找占用端口的进程
   sudo lsof -i :8080
   sudo netstat -tlnp | grep :8080
   ```

4. **内存不足**
   ```bash
   # 检查内存使用
   free -h
   ps aux | grep goblog
   
   # 调整内存限制
   # 编辑 .env 文件
   GOMAXPROCS=2
   ```

### 日志分析

```bash
# 错误日志
sudo journalctl -u goblog -p err

# 访问日志分析
sudo journalctl -u goblog | grep "HTTP" | tail -20

# 性能监控
sudo journalctl -u goblog | grep "slow query"
```

## 📈 性能优化

### 系统优化

```bash
# 调整文件描述符限制
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf

# 调整内核参数
echo "net.core.somaxconn = 65536" >> /etc/sysctl.conf
echo "net.ipv4.tcp_max_syn_backlog = 65536" >> /etc/sysctl.conf
sysctl -p
```

### 应用优化

编辑 `/opt/goblog/.env`：

```bash
# 启用性能分析
ENABLE_PPROF=true
PPROF_PORT=6060

# 调整连接数
DB_MAX_CONNECTIONS=20

# 启用指标监控
ENABLE_METRICS=true
METRICS_PORT=9090
```

## 📞 技术支持

如果在部署过程中遇到问题，请：

1. 检查日志文件
2. 查看系统资源使用情况
3. 确认配置文件正确性
4. 提交 Issue 到 GitHub 仓库

---

**注意**: 请根据实际环境调整配置参数，特别是在生产环境中部署时。