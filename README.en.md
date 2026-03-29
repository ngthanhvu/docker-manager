# Docker Manager - Desktop Application

[![Stars](https://img.shields.io/github/stars/ngthanhvu/docker-manager?style=for-the-badge)](https://github.com/ngthanhvu/docker-manager/stargazers)
[![Forks](https://img.shields.io/github/forks/ngthanhvu/docker-manager?style=for-the-badge)](https://github.com/ngthanhvu/docker-manager/forks)
[![License](https://img.shields.io/github/license/ngthanhvu/docker-manager?style=for-the-badge)](./LICENSE)
[![Last Commit](https://img.shields.io/github/last-commit/ngthanhvu/docker-manager?style=for-the-badge)](https://github.com/ngthanhvu/docker-manager/commits/main)
[![Tauri](https://img.shields.io/badge/Tauri-v2-24C8DB?style=for-the-badge&logo=tauri&logoColor=white)](https://tauri.app/)

A powerful desktop Docker management application with an intuitive interface, allowing you to monitor and control your entire Docker system from one place.

![Dock Manager Dashboard](./docs/screenshot.png)

## Key Features

- **Dashboard**: Monitor system health, resources, and Docker throughput in real-time
- **Containers**: Manage container lifecycle (start, stop, restart, remove)
- **Images**: View and manage local Docker images
- **Volumes**: Manage Docker volumes
- **Networks**: Configure and manage Docker networks
- **Compose**: Docker Compose support for multi-container applications
- **Monitoring**: Charts for network throughput, disk I/O, CPU and memory usage

## Technologies Used

| Component | Technology |
|-----------|------------|
| Frontend | Vue 3 + Vite |
| Backend API | Go (port `:8080`) |
| Desktop Shell | Tauri v2 |
| Styling | Bootstrap + Material Design |

## Project Structure

```
docker-manager/
├── frontend/          # Vue 3 UI with components
├── backend/           # Go REST API handling Docker operations
├── src-tauri/         # Tauri desktop configuration and source
├── docker-compose.yml # Dev environment configuration
├── docker-compose.prod.yml # Production environment configuration
├── run-dev.sh         # Dev environment script
└── run-prod.sh        # Production environment script
```

## Requirements

| Requirement | Version | Notes |
|-------------|---------|-------|
| OS | Windows 10/11 | |
| Docker | Docker Desktop or Docker Engine | Must be running before opening the app |
| Node.js | LTS | With npm |
| Go | As per `backend/go.mod` | |
| Rust | Latest | For Tauri build |

## Running in Development

Two ways to run the development environment:

### Option 1: Run Directly (without Docker)

Open 2 terminals at the project root:

1. **Run backend (Go):**
```powershell
cd backend
go mod download
go run .
```

2. **Run Tauri app:**
```powershell
cd ..
npm --prefix frontend install
npx tauri dev
```

> **Notes:**
> - Frontend dev server runs at `http://localhost:5173` (via Tauri's `beforeDevCommand`)
> - Backend API at `http://localhost:8080`

### Option 2: Run with Docker

```bash
./run-dev.sh
```

Services will run at:
- Frontend: `http://localhost:5173`
- Backend: `http://localhost:8080`

## Building for Release

To build the application into an installer for end users:

```powershell
npm --prefix frontend install
npx tauri build
```

**Build output:**

| Path | File Type | Description |
|------|-----------|-------------|
| `src-tauri/target/release/bundle/msi/` | `.msi` | Windows Installer |
| `src-tauri/target/release/bundle/nsis/` | `.exe` | NSIS Installer (if available) |

## Installation for End Users

1. Send the installer file from the `bundle` folder (`.msi` or `.exe`) to the user
2. User runs the installer to install the application
3. **Important:** Ensure Docker is running before opening the application

## Production Environment with Docker

The project supports running production environment with Docker:

| File | Description |
|------|-------------|
| `docker-compose.yml` | Development configuration |
| `docker-compose.prod.yml` | Production configuration |
| `backend/Dockerfile` | Dockerfile for backend (dev) |
| `backend/Dockerfile.prod` | Dockerfile for backend (prod) |
| `frontend/Dockerfile` | Dockerfile for frontend (dev) |
| `frontend/Dockerfile.prod` | Dockerfile for frontend (prod) |

### Running Production Environment Locally

```bash
./run-prod.sh up
```

**Common commands:**

```bash
./run-prod.sh down      # Stop and remove containers
./run-prod.sh logs      # View logs for all services
./run-prod.sh logs backend  # View backend logs only
./run-prod.sh restart   # Restart all services
```

## Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `identifier must be unique` | Duplicate `identifier` in `tauri.conf.json` | Change `identifier` to a different value (cannot be `com.tauri.dev`) |
| Icon/taskbar not updating | Windows cache | Close app, reopen, or unpin/pin shortcut on taskbar |
| Cannot connect to Docker daemon | Docker not running | Start Docker Desktop or Docker Service |
| Port 8080 already in use | Port occupied by another app | Change port in `backend/main.go` or stop the app using the port |

---

## Build and Push Docker Image to Docker Hub

Guide to building and pushing images to Docker Hub for deployment across environments.

### Step 1: Login to Docker Hub

```bash
docker login
```

Enter your username and password/token.

### Step 2: Build and Push Image

```bash
./run-prod.sh push <dockerhub_username> <repo_prefix> [service] [tag]
```

**Parameters:**

| Parameter | Required | Default | Description |
|-----------|----------|---------|-------------|
| `dockerhub_username` | ✅ | - | Your Docker Hub username |
| `repo_prefix` | ✅ | - | Repository name prefix (e.g., `docker-manager`) |
| `service` | ❌ | `all` | Service to push: `backend`, `frontend`, or `all` |
| `tag` | ❌ | `latest` | Version tag for the image |

**Examples:**

```bash
# Push both frontend and backend with tag v1.0.0
./run-prod.sh push yourname docker-manager all v1.0.0

# Push backend only with latest tag
./run-prod.sh push yourname docker-manager backend latest

# Push frontend only
./run-prod.sh push yourname docker-manager frontend v1.0.0
```

**Image format after push:**

```
<dockerhub_username>/<repo_prefix>-backend:<tag>
<dockerhub_username>/<repo_prefix>-frontend:<tag>
```

Example: `yourname/docker-manager-backend:v1.0.0`

### Step 3: Pull and Run Images from Docker Hub

```bash
# Pull images from Docker Hub
docker pull yourname/docker-manager-backend:v1.0.0
docker pull yourname/docker-manager-frontend:v1.0.0

# Run with docker-compose
docker-compose -f docker-compose.prod.yml up -d
```

---

## Languages / Ngôn ngữ / 语言

- [English](README.en.md)
- [Tiếng Việt](README.md)
- [中文](README.zh.md)
