# PMS - 个人管理系统

一个基于 Go 语言开发的个人管理系统，集成了财务管理、习惯追踪、任务管理、日记记录等功能。

## 功能特性

### 🏠 仪表板
- 统一的个人信息管理界面
- 快速访问各个功能模块
- 数据概览和统计

### 💰 财务管理
- 收入和支出记录
- 分类管理（支持自定义分类）
- 预算目标设置
- 财务数据导出

### 🎯 习惯追踪
- 日常习惯记录
- 连续打卡统计
- 进度可视化
- 频率设置（每日/每周）

### ✅ 任务管理
- 待办事项列表
- 任务状态管理
- 打卡功能
- 完成度统计

### 📔 日记记录
- 每日日记撰写
- 天气和心情记录
- 日记编辑和删除
- 历史记录查看

### 🏆 成就系统
- 徽章和成就解锁
- 使用进度激励
- 个人成长记录

### 👥 用户系统
- 用户注册和登录
- 多用户支持
- 管理员后台
- 数据隔离

## 技术栈

- **后端**: Go 1.25.5
- **数据库**: MySQL
- **Web框架**: 标准库 net/http
- **模板引擎**: Go html/template
- **依赖管理**: Go Modules

## 项目结构

```
PMS/
├── main.go              # 主程序入口
├── go.mod               # Go 模块文件
├── config.json          # 配置文件
├── init_database.sql    # 数据库初始化脚本
├── auth/                # 认证相关
├── config/              # 配置管理
├── db/                  # 数据库操作
├── handlers/            # HTTP 处理器
├── models/              # 数据模型
├── templates/           # HTML 模板
└── utils/               # 工具函数
```

## 快速开始

### 1. 环境要求
- Go 1.25.5 或更高版本
- MySQL 5.7 或更高版本

### 2. 安装依赖
```bash
go mod download
```

### 3. 数据库配置
- 创建 MySQL 数据库
- 修改 `config.json` 中的数据库连接信息
- 运行 `init_database.sql` 初始化数据库表

### 4. 启动应用
```bash
go run main.go
```

应用启动后会自动打开浏览器，访问 `http://localhost:8081`

### 5. 首次配置
- 首次访问会进入配置页面
- 设置数据库连接信息
- 创建管理员账户
- 完成初始化后即可使用

## 配置说明

`config.json` 配置文件包含以下选项：

```json
{
  "mysql": {
    "host": "localhost",
    "port": "3306",
    "user": "root",
    "password": "your_password",
    "database": "goblog"
  },
  "server": {
    "port": "8081"
  },
  "admin": {
    "username": "admin",
    "password": "admin"
  },
  "initialized": true
}
```

## API 路由

### 认证相关
- `GET/POST /login` - 用户登录
- `GET/POST /register` - 用户注册
- `GET /logout` - 用户登出

### 主要功能
- `GET /` - 仪表板
- `GET /finance` - 财务管理
- `GET /habits` - 习惯追踪
- `GET /todos` - 任务管理
- `GET /diary` - 日记记录

### 管理后台
- `GET /admin` - 管理员仪表板
- `GET /admin/users` - 用户管理
- `GET /admin/data` - 数据管理

## 开发说明

### 数据模型
主要的数据模型包括：
- `User` - 用户信息
- `Transaction` - 财务交易记录
- `Category` - 分类信息
- `Habit` - 习惯记录
- `Todo` - 任务信息
- `Diary` - 日记内容
- `Badge` - 徽章成就

### 中间件
- `AuthMiddleware` - 用户认证中间件
- `AdminAuthMiddleware` - 管理员认证中间件

### 数据库操作
使用 `database/sql` 包进行数据库操作，支持：
- 连接池管理
- 事务处理
- 预编译语句

## 部署

### 编译部署
```bash
go build -o pms main.go
./pms
```

### Docker 部署
```dockerfile
FROM golang:1.25.5-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o pms main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/pms .
EXPOSE 8081
CMD ["./pms"]
```

## 许可证

本项目采用 MIT 许可证。

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

## 更新日志

### v1.0.0
- 初始版本发布
- 完整的个人管理功能
- 用户认证系统
- 管理员后台