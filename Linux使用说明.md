# 自律人生 Linux 版使用说明

## 🐧 支持的Linux发行版

- Ubuntu 18.04+ / Debian 10+
- CentOS 7+ / RHEL 7+
- Fedora 30+
- Arch Linux
- openSUSE Leap 15.1+
- 其他主流Linux发行版

## 🚀 快速启动

### 方法一：使用启动脚本（推荐）

1. **下载并解压程序**
```bash
tar -xzf life-management-linux.tar.gz
cd life-management
```

2. **使用增强启动脚本**
```bash
chmod +x 启动.sh
./启动.sh
```

3. **使用简单启动脚本**
```bash
chmod +x 简单启动.sh
./简单启动.sh
```

### 方法二：直接运行

1. **添加执行权限**
```bash
chmod +x 自律人生-linux
```

2. **直接运行**
```bash
./自律人生-linux
```

3. **访问应用**
打开浏览器访问：http://localhost:8080

## 📋 系统要求

### 最低要求
- **CPU**: x86_64架构
- **内存**: 512MB RAM
- **磁盘**: 50MB可用空间
- **网络**: 无需网络连接（离线使用）

### 推荐配置
- **CPU**: 2核心以上
- **内存**: 1GB RAM以上
- **磁盘**: 100MB以上可用空间

## 🔧 高级配置

### 作为系统服务运行

1. **创建系统用户**
```bash
sudo useradd -r -s /bin/false life-user
```

2. **复制程序到系统目录**
```bash
sudo mkdir -p /opt/life-management
sudo cp 自律人生-linux /opt/life-management/
sudo cp -r data /opt/life-management/  # 如果已有数据
sudo cp -r templates /opt/life-management/  # 如果有外部模板
sudo chown -R life-user:life-user /opt/life-management
```

3. **安装systemd服务**
```bash
sudo cp life-management.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable life-management
sudo systemctl start life-management
```

4. **查看服务状态**
```bash
sudo systemctl status life-management
sudo journalctl -u life-management -f
```

### 反向代理配置

#### Nginx配置
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

#### Apache配置
```apache
<VirtualHost *:80>
    ServerName your-domain.com
    ProxyPreserveHost On
    ProxyRequests Off
    ProxyPass / http://localhost:8080/
    ProxyPassReverse / http://localhost:8080/
</VirtualHost>
```

## 🔒 安全配置

### 防火墙设置

#### UFW (Ubuntu)
```bash
sudo ufw allow 8080/tcp
sudo ufw reload
```

#### firewalld (CentOS/RHEL)
```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload
```

#### iptables
```bash
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
sudo iptables-save > /etc/iptables/rules.v4
```

### AppArmor/SELinux
如果启用了AppArmor或SELinux，可能需要配置安全策略：

```bash
# AppArmor
sudo aa-complain /opt/life-management/自律人生-linux

# SELinux
sudo setsebool -P httpd_can_network_connect 1
```

## 🛠 故障排除

### 常见问题

**Q: 提示权限被拒绝？**
```bash
# 添加执行权限
chmod +x 自律人生-linux 启动.sh 简单启动.sh

# 检查文件所有者
ls -la
```

**Q: 端口8080被占用？**
```bash
# 查看占用端口的进程
sudo netstat -tulpn | grep :8080
sudo ss -tulpn | grep :8080

# 终止占用进程
sudo kill -9 <PID>
```

**Q: 无法创建数据目录？**
```bash
# 检查目录权限
ls -ld data/
sudo chown $USER:$USER data/
sudo chmod 755 data/
```

**Q: 防火墙阻止访问？**
```bash
# 临时关闭防火墙测试
sudo ufw disable
sudo firewall-cmd --stop

# 或者只开放8080端口
sudo ufw allow 8080
sudo firewall-cmd --add-port=8080/tcp --permanent
```

**Q: 程序崩溃或无响应？**
```bash
# 查看系统日志
dmesg | tail -20
journalctl -xe

# 查看资源使用
top -p $(pgrep 自律人生-linux)
htop

# 检查磁盘空间
df -h
```

### 调试模式

如需详细调试信息，可以设置环境变量：

```bash
export DEBUG=true
./自律人生-linux
```

### 性能优化

#### 内存优化
```bash
# 限制内存使用
systemctl set-property life-management.service MemoryMax=256M

# 或者使用ulimit
ulimit -v 262144
./自律人生-linux
```

#### CPU优化
```bash
# 设置CPU亲和性
taskset -c 0,1 ./自律人生-linux

# 设置进程优先级
nice -n 10 ./自律人生-linux
```

## 📊 监控和日志

### 日志查看
```bash
# 实时查看程序输出
./自律人生-linux 2>&1 | tee life-management.log

# 使用logrotate管理日志
sudo nano /etc/logrotate.d/life-management
```

### 系统监控
```bash
# 监控进程状态
watch -n 1 'ps aux | grep 自律人生-linux'

# 监控网络连接
watch -n 1 'netstat -an | grep :8080'

# 监控资源使用
htop -p $(pgrep 自律人生-linux)
```

## 🔄 自动化脚本

### 启动脚本模板
```bash
#!/bin/bash
# /usr/local/bin/start-life-management

cd /opt/life-management
if ! pgrep -f "自律人生-linux" > /dev/null; then
    echo "启动自律人生..."
    ./自律人生-linux &
else
    echo "程序已在运行"
fi
```

### 备份脚本
```bash
#!/bin/bash
# /usr/local/bin/backup-life-management

BACKUP_DIR="/backup/life-management"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p "$BACKUP_DIR"
tar -czf "$BACKUP_DIR/life-management_$DATE.tar.gz" -C /opt/life-management data/

echo "备份完成: $BACKUP_DIR/life-management_$DATE.tar.gz"
```

## 📱 移动端访问

在局域网内，其他设备可以通过以下方式访问：

1. **查看服务器IP地址**
```bash
ip addr show
# 或者
hostname -I
```

2. **在移动设备访问**
```
http://[服务器IP]:8080
```

3. **确保防火墙允许局域网访问**
```bash
sudo ufw allow from 192.168.0.0/24 to any port 8080
```

---

**享受在Linux上的自律生活管理体验！** 🐧✨