# Docker Hub App (Tauri + Vue + Go)

Desktop app quản lý Docker (containers, images, volumes, networks, compose) với:
- Frontend: Vue 3 + Vite
- Backend API: Go (`:8080`)
- Desktop shell: Tauri v2

## Cấu trúc dự án

- `frontend/`: giao diện Vue
- `backend/`: API Go thao tác Docker
- `src-tauri/`: cấu hình và mã Tauri desktop

## Yêu cầu môi trường

- Windows 10/11
- Docker Desktop (hoặc Docker Engine) đang chạy
- Node.js LTS + npm
- Go (theo `backend/go.mod`)
- Rust toolchain (cho Tauri build)

## Chạy dev

Mở 2 terminal tại thư mục gốc project:

1. Chạy backend (Go):
```powershell
cd backend
go mod download
go run .
```

2. Chạy app Tauri:
```powershell
cd ..
npm --prefix frontend install
npx tauri dev
```

Ghi chú:
- Frontend sẽ chạy dev server tại `http://localhost:5173` (do Tauri tự gọi qua `beforeDevCommand`).
- Backend API ở `http://localhost:8080`.

## Build bản phát hành

Tại thư mục gốc project:

```powershell
npm --prefix frontend install
npx tauri build
```

Output nằm tại:
- `src-tauri/target/release/bundle/msi/` (installer `.msi`)
- có thể có thêm `src-tauri/target/release/bundle/nsis/` (installer `.exe`)

## Cài cho người dùng cuối

1. Gửi file installer trong thư mục `bundle` (`.msi` hoặc `.exe`).
2. Người dùng chạy installer để cài app.
3. Đảm bảo máy người dùng có Docker đang chạy trước khi mở app.

## Một số lỗi thường gặp

- Lỗi `identifier must be unique` khi build:
  - Kiểm tra `src-tauri/tauri.conf.json` và đảm bảo `identifier` không phải `com.tauri.dev`.
- Icon/taskbar chưa cập nhật ngay:
  - Đóng app mở lại, hoặc unpin/pin lại shortcut trên taskbar.

## Docker (Dev/Prod) và Push Docker Hub

Project dùng các file:
- Dev compose: `docker-compose.yml`
- Prod compose: `docker-compose.prod.yml`
- Dev Dockerfile: `backend/Dockerfile.dev`, `frontend/Dockerfile.dev`
- Prod Dockerfile: `backend/Dockerfile.prod`, `frontend/Dockerfile.prod`

### 1. Chạy môi trường dev bằng Docker

```bash
./run-dev.sh
```

Service chạy:
- Frontend: `http://localhost:5173`
- Backend: `http://localhost:8080`

### 2. Chạy môi trường prod local

```bash
./run-prod.sh up
```

Các lệnh hay dùng:

```bash
./run-prod.sh down
./run-prod.sh logs
./run-prod.sh logs backend
./run-prod.sh restart
```

### 3. Build và push image lên Docker Hub

1. Đăng nhập Docker Hub:
```bash
docker login
```

2. Build + push:
```bash
./run-prod.sh push <dockerhub_username> <repo_prefix> [service] [tag]
```

Trong đó:
- `service`: `backend` | `frontend` | `all` (mặc định `all`)
- `tag`: mặc định `latest`

Ví dụ:
```bash
./run-prod.sh push yourname docker-manager all v1.0.0
```

Image được push theo format:
- `<dockerhub_username>/<repo_prefix>-backend:<tag>`
- `<dockerhub_username>/<repo_prefix>-frontend:<tag>`
