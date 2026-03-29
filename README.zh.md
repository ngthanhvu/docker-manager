# Docker Manager - 桌面应用程序

[![Stars](https://img.shields.io/github/stars/ngthanhvu/docker-manager?style=for-the-badge)](https://github.com/ngthanhvu/docker-manager/stargazers)
[![Forks](https://img.shields.io/github/forks/ngthanhvu/docker-manager?style=for-the-badge)](https://github.com/ngthanhvu/docker-manager/forks)
[![License](https://img.shields.io/github/license/ngthanhvu/docker-manager?style=for-the-badge)](./LICENSE)
[![Last Commit](https://img.shields.io/github/last-commit/ngthanhvu/docker-manager?style=for-the-badge)](https://github.com/ngthanhvu/docker-manager/commits/main)
[![Tauri](https://img.shields.io/badge/Tauri-v2-24C8DB?style=for-the-badge&logo=tauri&logoColor=white)](https://tauri.app/)

一款功能强大的桌面 Docker 管理应用程序，具有直观的界面，让您能够从一个地方监控和控制整个 Docker 系统。

![Dock Manager Dashboard](./docs/screenshot.png)

## 主要功能

- **仪表板**: 实时监控 Docker 系统健康、资源和吞吐量
- **容器**: 管理容器生命周期（启动、停止、重启、删除）
- **镜像**: 查看和管理本地 Docker 镜像
- **数据卷**: 管理 Docker 数据卷
- **网络**: 配置和管理 Docker 网络
- **Compose**: 支持多容器应用的 Docker Compose
- **监控**: 网络吞吐量、磁盘 I/O、CPU 和内存使用率图表

## 使用的技术

| 组件 | 技术 |
|------|------|
| 前端 | Vue 3 + Vite |
| 后端 API | Go（端口 `:8080`） |
| 桌面 Shell | Tauri v2 |
| 样式 | Bootstrap + Material Design |

## 项目结构

```
docker-manager/
── frontend/          # Vue 3 界面和组件
├── backend/           # Go REST API 处理 Docker 操作
├── src-tauri/         # Tauri 桌面配置和源码
├── docker-compose.yml # 开发环境配置
├── docker-compose.prod.yml # 生产环境配置
├── run-dev.sh         # 开发环境脚本
└── run-prod.sh        # 生产环境脚本
```

## 系统要求

| 要求 | 版本 | 备注 |
|------|------|------|
| 操作系统 | Windows 10/11 | |
| Docker | Docker Desktop 或 Docker Engine | 必须在打开应用前启动 |
| Node.js | LTS | 含 npm |
| Go | 参考 `backend/go.mod` | |
| Rust | 最新版 | 用于 Tauri 构建 |

## 开发环境运行

有两种方式运行开发环境：

### 方式一：直接运行（不使用 Docker）

在项目根目录打开 2 个终端：

1. **运行后端（Go）：**
```powershell
cd backend
go mod download
go run .
```

2. **运行 Tauri 应用：**
```powershell
cd ..
npm --prefix frontend install
npx tauri dev
```

> **注意：**
> - 前端开发服务器运行在 `http://localhost:5173`（通过 Tauri 的 `beforeDevCommand`）
> - 后端 API 在 `http://localhost:8080`

### 方式二：使用 Docker 运行

```bash
./run-dev.sh
```

服务运行地址：
- 前端：`http://localhost:5173`
- 后端：`http://localhost:8080`

## 构建发布版本

将应用构建为最终用户的安装文件：

```powershell
npm --prefix frontend install
npx tauri build
```

**构建输出：**

| 路径 | 文件类型 | 描述 |
|------|----------|------|
| `src-tauri/target/release/bundle/msi/` | `.msi` | Windows 安装程序 |
| `src-tauri/target/release/bundle/nsis/` | `.exe` | NSIS 安装程序（如有） |

## 最终用户安装

1. 将 `bundle` 文件夹中的安装文件（`.msi` 或 `.exe`）发送给用户
2. 用户运行安装文件安装应用程序
3. **重要：** 确保 Docker 在打开应用前正在运行

## Docker 生产环境

项目支持使用 Docker 运行生产环境：

| 文件 | 描述 |
|------|------|
| `docker-compose.yml` | 开发环境配置 |
| `docker-compose.prod.yml` | 生产环境配置 |
| `backend/Dockerfile` | 后端 Dockerfile（开发） |
| `backend/Dockerfile.prod` | 后端 Dockerfile（生产） |
| `frontend/Dockerfile` | 前端 Dockerfile（开发） |
| `frontend/Dockerfile.prod` | 前端 Dockerfile（生产） |

### 本地运行生产环境

```bash
./run-prod.sh up
```

**常用命令：**

```bash
./run-prod.sh down      # 停止并删除容器
./run-prod.sh logs      # 查看所有服务日志
./run-prod.sh logs backend  # 仅查看后端日志
./run-prod.sh restart   # 重启所有服务
```

## 常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| `identifier must be unique` | `tauri.conf.json` 中的 `identifier` 重复 | 将 `identifier` 改为其他值（不能是 `com.tauri.dev`） |
| 图标/任务栏不更新 | Windows 缓存 | 关闭应用后重新打开，或在任务栏取消固定后重新固定 |
| Cannot connect to Docker daemon | Docker 未运行 | 启动 Docker Desktop 或 Docker Service |
| Port 8080 already in use | 端口被其他应用占用 | 在 `backend/main.go` 中更改端口或停止占用端口的应用 |

---

## 构建并推送 Docker 镜像到 Docker Hub

构建并推送镜像到 Docker Hub 以便在多个环境部署。

### 步骤 1：登录 Docker Hub

```bash
docker login
```

输入您的用户名和密码/令牌。

### 步骤 2：构建并推送镜像

```bash
./run-prod.sh push <dockerhub_username> <repo_prefix> [service] [tag]
```

**参数：**

| 参数 | 必填 | 默认值 | 描述 |
|------|------|--------|------|
| `dockerhub_username` | ✅ | - | 您的 Docker Hub 用户名 |
| `repo_prefix` | ✅ | - | 仓库名称前缀（例如：`docker-manager`） |
| `service` | ❌ | `all` | 要推送的服务：`backend`、`frontend` 或 `all` |
| `tag` | ❌ | `latest` | 镜像版本标签 |

**示例：**

```bash
# 推送前端和后端，标签 v1.0.0
./run-prod.sh push yourname docker-manager all v1.0.0

# 仅推送后端，标签 latest
./run-prod.sh push yourname docker-manager backend latest

# 仅推送前端
./run-prod.sh push yourname docker-manager frontend v1.0.0
```

**推送后的镜像格式：**

```
<dockerhub_username>/<repo_prefix>-backend:<tag>
<dockerhub_username>/<repo_prefix>-frontend:<tag>
```

示例：`yourname/docker-manager-backend:v1.0.0`

### 步骤 3：从 Docker Hub 拉取并运行镜像

```bash
# 从 Docker Hub 拉取镜像
docker pull yourname/docker-manager-backend:v1.0.0
docker pull yourname/docker-manager-frontend:v1.0.0

# 使用 docker-compose 运行
docker-compose -f docker-compose.prod.yml up -d
```

---

## Languages / Ngôn ngữ / 语言

- [English](README.en.md)
- [Tiếng Việt](README.md)
- [中文](README.zh.md)
