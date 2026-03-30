#!/bin/bash

set -e

APP_DIR="/opt/docker-manager"

DOCKERHUB_NAMESPACE="${DOCKERHUB_NAMESPACE:-ngthanhvu}"
REPO_PREFIX="${REPO_PREFIX:-docker-manager}"
IMAGE_TAG="${IMAGE_TAG:-}"

BACKEND_IMAGE=""
FRONTEND_IMAGE=""

DEFAULT_PORT=8088
LANGUAGE="en"

tr() {
    local key="$1"
    case "$LANGUAGE:$key" in
        vi:need_root)
            echo "Trình cài đặt cần quyền root để ghi vào $APP_DIR và quản lý Docker."
            ;;
        vi:run_with_sudo)
            echo "Vui lòng chạy bằng sudo, ví dụ:"
            ;;
        vi:installer_title)
            echo " Docker Manager Installer"
            ;;
        vi:language_prompt)
            echo "Chọn ngôn ngữ / Choose language / 选择语言:"
            ;;
        vi:language_vi)
            echo "1) Tiếng Việt"
            ;;
        vi:language_en)
            echo "2) English"
            ;;
        vi:language_zh)
            echo "3) 中文"
            ;;
        vi:language_choose)
            echo "Chọn ngôn ngữ [1-3]: "
            ;;
        vi:menu_install)
            echo "1) Cài đặt Docker Manager"
            ;;
        vi:menu_uninstall)
            echo "2) Gỡ cài đặt Docker Manager"
            ;;
        vi:menu_choose)
            echo "Chọn tùy chọn [1-2]: "
            ;;
        vi:resolving_latest)
            echo "Đang lấy phiên bản mới nhất từ Docker Hub..."
            ;;
        vi:using_image_tag)
            echo "Sử dụng image tag:"
            ;;
        vi:backend_image)
            echo "Backend image:"
            ;;
        vi:frontend_image)
            echo "Frontend image:"
            ;;
        vi:installing)
            echo "Đang cài đặt Docker Manager..."
            ;;
        vi:docker_not_found)
            echo "Không tìm thấy Docker. Đang cài đặt..."
            ;;
        vi:compose_required)
            echo "Yêu cầu Docker Compose v2."
            ;;
        vi:enter_port)
            echo "Nhập cổng cho Docker Manager"
            ;;
        vi:port_in_use)
            echo "Cổng %s đang được sử dụng."
            ;;
        vi:choose_another_port)
            echo "Chọn cổng khác: "
            ;;
        vi:using_port)
            echo "Sử dụng cổng:"
            ;;
        vi:pulling_images)
            echo "Đang kéo image..."
            ;;
        vi:starting_app)
            echo "Đang khởi động Docker Manager..."
            ;;
        vi:installed_title)
            echo " Docker Manager Đã Được Cài Đặt"
            ;;
        vi:access_url)
            echo "URL truy cập:"
            ;;
        vi:install_directory)
            echo "Thư mục cài đặt:"
            ;;
        vi:not_installed)
            echo "Docker Manager chưa được cài đặt."
            ;;
        vi:confirm_uninstall)
            echo "Bạn có chắc muốn gỡ Docker Manager không? (y/n): "
            ;;
        vi:cancelled)
            echo "Đã hủy."
            ;;
        vi:stopping_containers)
            echo "Đang dừng containers..."
            ;;
        vi:removing_images)
            echo "Đang xóa images..."
            ;;
        vi:removing_install_dir)
            echo "Đang xóa thư mục cài đặt..."
            ;;
        vi:removed_success)
            echo "Đã gỡ Docker Manager thành công."
            ;;
        vi:invalid_option)
            echo "Lựa chọn không hợp lệ."
            ;;
        zh:need_root)
            echo "安装程序需要 root 权限来写入 $APP_DIR 并管理 Docker。"
            ;;
        zh:run_with_sudo)
            echo "请使用 sudo 运行，例如："
            ;;
        zh:installer_title)
            echo " Docker Manager 安装程序"
            ;;
        zh:language_prompt)
            echo "选择语言 / Choose language / Chọn ngôn ngữ:"
            ;;
        zh:language_vi)
            echo "1) Tiếng Việt"
            ;;
        zh:language_en)
            echo "2) English"
            ;;
        zh:language_zh)
            echo "3) 中文"
            ;;
        zh:language_choose)
            echo "选择语言 [1-3]: "
            ;;
        zh:menu_install)
            echo "1) 安装 Docker Manager"
            ;;
        zh:menu_uninstall)
            echo "2) 卸载 Docker Manager"
            ;;
        zh:menu_choose)
            echo "选择操作 [1-2]: "
            ;;
        zh:resolving_latest)
            echo "正在从 Docker Hub 获取最新版本..."
            ;;
        zh:using_image_tag)
            echo "使用的镜像标签:"
            ;;
        zh:backend_image)
            echo "后端镜像:"
            ;;
        zh:frontend_image)
            echo "前端镜像:"
            ;;
        zh:installing)
            echo "正在安装 Docker Manager..."
            ;;
        zh:docker_not_found)
            echo "未找到 Docker。正在安装..."
            ;;
        zh:compose_required)
            echo "需要 Docker Compose v2。"
            ;;
        zh:enter_port)
            echo "输入 Docker Manager 端口"
            ;;
        zh:port_in_use)
            echo "端口 %s 已被占用。"
            ;;
        zh:choose_another_port)
            echo "请选择其他端口: "
            ;;
        zh:using_port)
            echo "使用端口:"
            ;;
        zh:pulling_images)
            echo "正在拉取镜像..."
            ;;
        zh:starting_app)
            echo "正在启动 Docker Manager..."
            ;;
        zh:installed_title)
            echo " Docker Manager 安装完成"
            ;;
        zh:access_url)
            echo "访问地址:"
            ;;
        zh:install_directory)
            echo "安装目录:"
            ;;
        zh:not_installed)
            echo "Docker Manager 尚未安装。"
            ;;
        zh:confirm_uninstall)
            echo "确定要卸载 Docker Manager 吗？(y/n): "
            ;;
        zh:cancelled)
            echo "已取消。"
            ;;
        zh:stopping_containers)
            echo "正在停止容器..."
            ;;
        zh:removing_images)
            echo "正在删除镜像..."
            ;;
        zh:removing_install_dir)
            echo "正在删除安装目录..."
            ;;
        zh:removed_success)
            echo "Docker Manager 已成功卸载。"
            ;;
        zh:invalid_option)
            echo "无效选项。"
            ;;
        en:need_root|*:need_root)
            echo "This installer needs root privileges to write to $APP_DIR and manage Docker."
            ;;
        en:run_with_sudo|*:run_with_sudo)
            echo "Please run it with sudo, for example:"
            ;;
        en:installer_title|*:installer_title)
            echo " Docker Manager Installer"
            ;;
        en:language_prompt|*:language_prompt)
            echo "Choose language / Chọn ngôn ngữ / 选择语言:"
            ;;
        en:language_vi|*:language_vi)
            echo "1) Tiếng Việt"
            ;;
        en:language_en|*:language_en)
            echo "2) English"
            ;;
        en:language_zh|*:language_zh)
            echo "3) 中文"
            ;;
        en:language_choose|*:language_choose)
            echo "Choose language [1-3]: "
            ;;
        en:menu_install|*:menu_install)
            echo "1) Install Docker Manager"
            ;;
        en:menu_uninstall|*:menu_uninstall)
            echo "2) Uninstall Docker Manager"
            ;;
        en:menu_choose|*:menu_choose)
            echo "Choose option [1-2]: "
            ;;
        en:resolving_latest|*:resolving_latest)
            echo "Resolving latest version from Docker Hub..."
            ;;
        en:using_image_tag|*:using_image_tag)
            echo "Using image tag:"
            ;;
        en:backend_image|*:backend_image)
            echo "Backend image:"
            ;;
        en:frontend_image|*:frontend_image)
            echo "Frontend image:"
            ;;
        en:installing|*:installing)
            echo "Installing Docker Manager..."
            ;;
        en:docker_not_found|*:docker_not_found)
            echo "Docker not found. Installing..."
            ;;
        en:compose_required|*:compose_required)
            echo "Docker Compose v2 is required."
            ;;
        en:enter_port|*:enter_port)
            echo "Enter port for Docker Manager"
            ;;
        en:port_in_use|*:port_in_use)
            echo "Port %s is already in use."
            ;;
        en:choose_another_port|*:choose_another_port)
            echo "Choose another port: "
            ;;
        en:using_port|*:using_port)
            echo "Using port:"
            ;;
        en:pulling_images|*:pulling_images)
            echo "Pulling images..."
            ;;
        en:starting_app|*:starting_app)
            echo "Starting Docker Manager..."
            ;;
        en:installed_title|*:installed_title)
            echo " Docker Manager Installed"
            ;;
        en:access_url|*:access_url)
            echo "Access URL:"
            ;;
        en:install_directory|*:install_directory)
            echo "Install directory:"
            ;;
        en:not_installed|*:not_installed)
            echo "Docker Manager is not installed."
            ;;
        en:confirm_uninstall|*:confirm_uninstall)
            echo "Are you sure you want to uninstall Docker Manager? (y/n): "
            ;;
        en:cancelled|*:cancelled)
            echo "Cancelled."
            ;;
        en:stopping_containers|*:stopping_containers)
            echo "Stopping containers..."
            ;;
        en:removing_images|*:removing_images)
            echo "Removing images..."
            ;;
        en:removing_install_dir|*:removing_install_dir)
            echo "Removing install directory..."
            ;;
        en:removed_success|*:removed_success)
            echo "Docker Manager removed successfully."
            ;;
        en:invalid_option|*:invalid_option)
            echo "Invalid option."
            ;;
        *)
            echo "$key"
            ;;
    esac
}

printf "=================================\n"
printf "%s\n" "$(tr installer_title)"
printf "=================================\n\n"
printf "%s\n" "$(tr language_prompt)"
printf "%s\n" "$(tr language_vi)"
printf "%s\n" "$(tr language_en)"
printf "%s\n" "$(tr language_zh)"
read -p "$(tr language_choose)" LANGUAGE_OPTION

case "$LANGUAGE_OPTION" in
    1) LANGUAGE="vi" ;;
    2) LANGUAGE="en" ;;
    3) LANGUAGE="zh" ;;
    *) LANGUAGE="en" ;;
esac

if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
else
    if ! command -v sudo >/dev/null 2>&1; then
        echo "$(tr need_root)"
        echo "$(tr run_with_sudo)"
        echo "sudo bash <(curl -Ls https://raw.githubusercontent.com/ngthanhvu/docker-manager/refs/heads/main/install.sh)"
        exit 1
    fi
    SUDO="sudo"
fi

echo "================================="
echo "$(tr installer_title)"
echo "================================="
echo ""
echo "$(tr menu_install)"
echo "$(tr menu_uninstall)"
echo ""

read -p "$(tr menu_choose)" OPTION

resolve_latest_tag() {
    local repo_name="$1"
    local api_url="https://hub.docker.com/v2/namespaces/${DOCKERHUB_NAMESPACE}/repositories/${repo_name}/tags?page_size=100"
    local latest_tag=""

    if command -v curl >/dev/null 2>&1; then
        latest_tag=$(
            curl -fsSL "$api_url" 2>/dev/null \
                | grep -oE '"name"[[:space:]]*:[[:space:]]*"[^"]+"' \
                | sed -E 's/"name"[[:space:]]*:[[:space:]]*"([^"]+)"/\1/' \
                | grep -E '^v?[0-9]+(\.[0-9]+){0,3}([-.+][0-9A-Za-z.-]+)?$' \
                | sort -rV \
                | head -n 1
        )
    fi

    if [ -z "$latest_tag" ]; then
        latest_tag="latest"
    fi

    echo "$latest_tag"
}

set_image_refs() {
    local resolved_tag="${IMAGE_TAG:-}"
    if [ -z "$resolved_tag" ]; then
        echo "$(tr resolving_latest)"
        resolved_tag="$(resolve_latest_tag "${REPO_PREFIX}-frontend")"
    fi

    BACKEND_IMAGE="${DOCKERHUB_NAMESPACE}/${REPO_PREFIX}-backend:${resolved_tag}"
    FRONTEND_IMAGE="${DOCKERHUB_NAMESPACE}/${REPO_PREFIX}-frontend:${resolved_tag}"

    echo "$(tr using_image_tag) ${resolved_tag}"
    echo "$(tr backend_image) ${BACKEND_IMAGE}"
    echo "$(tr frontend_image) ${FRONTEND_IMAGE}"
}

# -------------------------
# INSTALL
# -------------------------

install_app() {

echo "$(tr installing)"

# check docker
if ! command -v docker &> /dev/null
then
    echo "$(tr docker_not_found)"
    curl -fsSL https://get.docker.com | $SUDO sh
fi

# check compose
if ! docker compose version &> /dev/null
then
    echo "$(tr compose_required)"
    exit 1
fi

set_image_refs

# choose port
read -p "$(tr enter_port) [${DEFAULT_PORT}]: " PORT
PORT=${PORT:-$DEFAULT_PORT}

# check port
check_port() {
    if lsof -i:$1 >/dev/null 2>&1; then
        return 1
    else
        return 0
    fi
}

while ! check_port $PORT; do
    printf "$(tr port_in_use)\n" "$PORT"
    read -p "$(tr choose_another_port)" PORT
done

echo "$(tr using_port) $PORT"

# create install dir
$SUDO mkdir -p "$APP_DIR"
cd "$APP_DIR"

# create compose file
$SUDO tee "$APP_DIR/docker-compose.yml" >/dev/null <<EOF
version: "3.8"

services:

  backend:
    image: $BACKEND_IMAGE
    container_name: docker-manager-backend
    restart: unless-stopped
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    ports:
      - "127.0.0.1:8080:8080"

  frontend:
    image: $FRONTEND_IMAGE
    container_name: docker-manager-frontend
    restart: unless-stopped
    depends_on:
      - backend
    ports:
      - "$PORT:80"
EOF

echo "$(tr pulling_images)"
$SUDO docker compose -f "$APP_DIR/docker-compose.yml" pull

echo "$(tr starting_app)"
$SUDO docker compose -f "$APP_DIR/docker-compose.yml" up -d

IP=$(hostname -I | awk '{print $1}')

echo ""
echo "================================="
echo "$(tr installed_title)"
echo "================================="
echo ""
echo "$(tr access_url)"
echo "http://$IP:$PORT"
echo ""
echo "$(tr install_directory)"
echo "$APP_DIR"
echo ""

}

# -------------------------
# UNINSTALL
# -------------------------

uninstall_app() {
if [ ! -d "$APP_DIR" ]; then
    echo "$(tr not_installed)"
    exit 0
fi

read -p "$(tr confirm_uninstall)" CONFIRM

if [ "$CONFIRM" != "y" ]; then
    echo "$(tr cancelled)"
    exit 0
fi

cd "$APP_DIR"

INSTALLED_BACKEND_IMAGE=$(sed -n '/backend:/,/frontend:/ s/^[[:space:]]*image:[[:space:]]*//p' "$APP_DIR/docker-compose.yml" | head -n 1)
INSTALLED_FRONTEND_IMAGE=$(sed -n '/frontend:/,$ s/^[[:space:]]*image:[[:space:]]*//p' "$APP_DIR/docker-compose.yml" | head -n 1)

echo "$(tr stopping_containers)"
$SUDO docker compose -f "$APP_DIR/docker-compose.yml" down

echo "$(tr removing_images)"
$SUDO docker image rm "${INSTALLED_BACKEND_IMAGE:-$BACKEND_IMAGE}" 2>/dev/null || true
$SUDO docker image rm "${INSTALLED_FRONTEND_IMAGE:-$FRONTEND_IMAGE}" 2>/dev/null || true

echo "$(tr removing_install_dir)"
$SUDO rm -rf "$APP_DIR"

echo ""
echo "$(tr removed_success)"
echo ""

}

# -------------------------
# MENU
# -------------------------

case $OPTION in
1)
    install_app
    ;;
2)
    uninstall_app
    ;;
*)
    echo "$(tr invalid_option)"
    ;;
esac
