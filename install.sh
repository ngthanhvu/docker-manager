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

# ─────────────────────────────────────────────
# COLORS & STYLES
# ─────────────────────────────────────────────

RESET="\033[0m"
BOLD="\033[1m"
DIM="\033[2m"

BLACK="\033[30m"
WHITE="\033[97m"
CYAN="\033[96m"
GREEN="\033[92m"
YELLOW="\033[93m"
RED="\033[91m"
BLUE="\033[94m"
MAGENTA="\033[95m"

BG_CYAN="\033[46m"
BG_WHITE="\033[107m"

# ─────────────────────────────────────────────
# ASCII ART BANNER
# ─────────────────────────────────────────────

show_banner() {
    clear
    echo ""
    printf "${CYAN}${BOLD}"
    echo "  ██████╗  ██████╗  ██████╗██╗  ██╗███████╗██████╗ "
    echo "  ██╔══██╗██╔═══██╗██╔════╝██║ ██╔╝██╔════╝██╔══██╗"
    echo "  ██║  ██║██║   ██║██║     █████╔╝ █████╗  ██████╔╝"
    echo "  ██║  ██║██║   ██║██║     ██╔═██╗ ██╔══╝  ██╔══██╗"
    echo "  ██████╔╝╚██████╔╝╚██████╗██║  ██╗███████╗██║  ██║"
    echo "  ╚═════╝  ╚═════╝  ╚═════╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝"
    printf "${RESET}"
    printf "${DIM}${WHITE}"
    echo "  ███╗   ███╗ █████╗ ███╗   ██╗ █████╗  ██████╗ ███████╗██████╗ "
    echo "  ████╗ ████║██╔══██╗████╗  ██║██╔══██╗██╔════╝ ██╔════╝██╔══██╗"
    echo "  ██╔████╔██║███████║██╔██╗ ██║███████║██║  ███╗█████╗  ██████╔╝"
    echo "  ██║╚██╔╝██║██╔══██║██║╚██╗██║██╔══██║██║   ██║██╔══╝  ██╔══██╗"
    echo "  ██║ ╚═╝ ██║██║  ██║██║ ╚████║██║  ██║╚██████╔╝███████╗██║  ██║"
    echo "  ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝"
    printf "${RESET}"
    echo ""
    printf "  ${DIM}v1.0  ·  by ngthanhvu${RESET}\n"
    echo ""
}

# ─────────────────────────────────────────────
# ARROW-KEY SELECTOR
# Usage: arrow_select "TITLE" "VAR_NAME" "opt1" "opt2" ...
# Sets the chosen value into VAR_NAME
# ─────────────────────────────────────────────

arrow_select() {
    local title="$1"
    local varname="$2"
    shift 2
    local options=("$@")
    local count=${#options[@]}
    local selected=0

    # Save cursor, hide it
    tput civis 2>/dev/null || true

    # Print title
    printf "  ${BOLD}${WHITE}${title}${RESET}\n\n"

    _render_menu() {
        for i in "${!options[@]}"; do
            if [ "$i" -eq "$selected" ]; then
                printf "  ${CYAN}${BOLD}▶  ${options[$i]}${RESET}\n"
            else
                printf "  ${DIM}   ${options[$i]}${RESET}\n"
            fi
        done
    }

    _render_menu

    while true; do
        # Read a key
        IFS= read -rsn1 key 2>/dev/null
        if [[ "$key" == $'\x1b' ]]; then
            IFS= read -rsn1 -t 0.1 key2 2>/dev/null
            IFS= read -rsn1 -t 0.1 key3 2>/dev/null
            if [[ "$key2" == "[" ]]; then
                case "$key3" in
                    "A") # Up
                        ((selected--)) || true
                        if [ "$selected" -lt 0 ]; then selected=$((count - 1)); fi
                        ;;
                    "B") # Down
                        ((selected++)) || true
                        if [ "$selected" -ge "$count" ]; then selected=0; fi
                        ;;
                esac
            fi
        elif [[ "$key" == "" ]]; then
            # Enter pressed
            break
        fi

        # Move cursor up to re-render
        for ((i=0; i<count; i++)); do
            tput cuu1 2>/dev/null || printf "\033[A"
        done

        _render_menu
    done

    tput cnorm 2>/dev/null || true
    echo ""
    eval "$varname=\"${options[$selected]}\""
}

# ─────────────────────────────────────────────
# TRANSLATIONS
# ─────────────────────────────────────────────

tr() {
    local key="$1"
    case "$LANGUAGE:$key" in
        vi:need_root)           echo "Trình cài đặt cần quyền root để ghi vào $APP_DIR và quản lý Docker." ;;
        vi:run_with_sudo)       echo "Vui lòng chạy bằng sudo, ví dụ:" ;;
        vi:menu_install)        echo "Cài đặt Docker Manager" ;;
        vi:menu_uninstall)      echo "Gỡ cài đặt Docker Manager" ;;
        vi:resolving_latest)    echo "Đang lấy phiên bản mới nhất từ Docker Hub..." ;;
        vi:using_image_tag)     echo "Sử dụng image tag:" ;;
        vi:backend_image)       echo "Backend image:" ;;
        vi:frontend_image)      echo "Frontend image:" ;;
        vi:installing)          echo "Đang cài đặt Docker Manager..." ;;
        vi:docker_not_found)    echo "Không tìm thấy Docker. Đang cài đặt..." ;;
        vi:compose_required)    echo "Yêu cầu Docker Compose v2." ;;
        vi:enter_port)          echo "Nhập cổng cho Docker Manager" ;;
        vi:port_in_use)         echo "Cổng %s đang được sử dụng." ;;
        vi:choose_another_port) echo "Chọn cổng khác: " ;;
        vi:using_port)          echo "Sử dụng cổng:" ;;
        vi:pulling_images)      echo "Đang kéo image..." ;;
        vi:starting_app)        echo "Đang khởi động Docker Manager..." ;;
        vi:installed_title)     echo "Docker Manager Đã Được Cài Đặt" ;;
        vi:access_url)          echo "URL truy cập:" ;;
        vi:install_directory)   echo "Thư mục cài đặt:" ;;
        vi:not_installed)       echo "Docker Manager chưa được cài đặt." ;;
        vi:confirm_uninstall)   echo "Bạn có chắc muốn gỡ Docker Manager không? (y/n): " ;;
        vi:cancelled)           echo "Đã hủy." ;;
        vi:stopping_containers) echo "Đang dừng containers..." ;;
        vi:removing_images)     echo "Đang xóa images..." ;;
        vi:removing_install_dir)echo "Đang xóa thư mục cài đặt..." ;;
        vi:removed_success)     echo "Đã gỡ Docker Manager thành công." ;;
        vi:invalid_option)      echo "Lựa chọn không hợp lệ." ;;
        vi:step_lang)           echo "Ngôn ngữ" ;;
        vi:step_action)         echo "Hành động" ;;
        vi:step_config)         echo "Cấu hình" ;;
        vi:step_install)        echo "Cài đặt" ;;

        zh:need_root)           echo "安装程序需要 root 权限来写入 $APP_DIR 并管理 Docker。" ;;
        zh:run_with_sudo)       echo "请使用 sudo 运行，例如：" ;;
        zh:menu_install)        echo "安装 Docker Manager" ;;
        zh:menu_uninstall)      echo "卸载 Docker Manager" ;;
        zh:resolving_latest)    echo "正在从 Docker Hub 获取最新版本..." ;;
        zh:using_image_tag)     echo "使用的镜像标签:" ;;
        zh:backend_image)       echo "后端镜像:" ;;
        zh:frontend_image)      echo "前端镜像:" ;;
        zh:installing)          echo "正在安装 Docker Manager..." ;;
        zh:docker_not_found)    echo "未找到 Docker。正在安装..." ;;
        zh:compose_required)    echo "需要 Docker Compose v2。" ;;
        zh:enter_port)          echo "输入 Docker Manager 端口" ;;
        zh:port_in_use)         echo "端口 %s 已被占用。" ;;
        zh:choose_another_port) echo "请选择其他端口: " ;;
        zh:using_port)          echo "使用端口:" ;;
        zh:pulling_images)      echo "正在拉取镜像..." ;;
        zh:starting_app)        echo "正在启动 Docker Manager..." ;;
        zh:installed_title)     echo "Docker Manager 安装完成" ;;
        zh:access_url)          echo "访问地址:" ;;
        zh:install_directory)   echo "安装目录:" ;;
        zh:not_installed)       echo "Docker Manager 尚未安装。" ;;
        zh:confirm_uninstall)   echo "确定要卸载 Docker Manager 吗？(y/n): " ;;
        zh:cancelled)           echo "已取消。" ;;
        zh:stopping_containers) echo "正在停止容器..." ;;
        zh:removing_images)     echo "正在删除镜像..." ;;
        zh:removing_install_dir)echo "正在删除安装目录..." ;;
        zh:removed_success)     echo "Docker Manager 已成功卸载。" ;;
        zh:invalid_option)      echo "无效选项。" ;;
        zh:step_lang)           echo "语言" ;;
        zh:step_action)         echo "操作" ;;
        zh:step_config)         echo "配置" ;;
        zh:step_install)        echo "安装" ;;

        *:need_root)            echo "This installer needs root privileges to write to $APP_DIR and manage Docker." ;;
        *:run_with_sudo)        echo "Please run it with sudo, for example:" ;;
        *:menu_install)         echo "Install Docker Manager" ;;
        *:menu_uninstall)       echo "Uninstall Docker Manager" ;;
        *:resolving_latest)     echo "Resolving latest version from Docker Hub..." ;;
        *:using_image_tag)      echo "Using image tag:" ;;
        *:backend_image)        echo "Backend image:" ;;
        *:frontend_image)       echo "Frontend image:" ;;
        *:installing)           echo "Installing Docker Manager..." ;;
        *:docker_not_found)     echo "Docker not found. Installing..." ;;
        *:compose_required)     echo "Docker Compose v2 is required." ;;
        *:enter_port)           echo "Enter port for Docker Manager" ;;
        *:port_in_use)          echo "Port %s is already in use." ;;
        *:choose_another_port)  echo "Choose another port: " ;;
        *:using_port)           echo "Using port:" ;;
        *:pulling_images)       echo "Pulling images..." ;;
        *:starting_app)         echo "Starting Docker Manager..." ;;
        *:installed_title)      echo "Docker Manager Installed" ;;
        *:access_url)           echo "Access URL:" ;;
        *:install_directory)    echo "Install directory:" ;;
        *:not_installed)        echo "Docker Manager is not installed." ;;
        *:confirm_uninstall)    echo "Are you sure you want to uninstall Docker Manager? (y/n): " ;;
        *:cancelled)            echo "Cancelled." ;;
        *:stopping_containers)  echo "Stopping containers..." ;;
        *:removing_images)      echo "Removing images..." ;;
        *:removing_install_dir) echo "Removing install directory..." ;;
        *:removed_success)      echo "Docker Manager removed successfully." ;;
        *:invalid_option)       echo "Invalid option." ;;
        *:step_lang)            echo "Language" ;;
        *:step_action)          echo "Action" ;;
        *:step_config)          echo "Configure" ;;
        *:step_install)         echo "Install" ;;
        *)                      echo "$key" ;;
    esac
}

# ─────────────────────────────────────────────
# STEP INDICATOR
# ─────────────────────────────────────────────

show_step() {
    local current="$1"
    local total="$2"
    local label="$3"
    printf "  ${DIM}Step ${current}/${total}${RESET}  ${CYAN}${BOLD}${label}${RESET}\n"
    printf "  ${DIM}$(printf '─%.0s' {1..44})${RESET}\n\n"
}

# ─────────────────────────────────────────────
# LOG HELPERS
# ─────────────────────────────────────────────

log_info()    { printf "  ${CYAN}❯${RESET}  %s\n" "$1"; }
log_success() { printf "  ${GREEN}✔${RESET}  %s\n" "$1"; }
log_warn()    { printf "  ${YELLOW}⚠${RESET}  %s\n" "$1"; }
log_error()   { printf "  ${RED}✖${RESET}  %s\n" "$1"; }
log_dim()     { printf "  ${DIM}   %s${RESET}\n" "$1"; }

# ─────────────────────────────────────────────
# STEP 1 — Language
# ─────────────────────────────────────────────

show_banner
show_step 1 3 "$(tr step_lang)"

arrow_select "$(tr step_lang)" LANG_CHOICE \
    "[VI]  Tiếng Việt" \
    "[EN]  English" \
    "[ZH]  中文"

case "$LANG_CHOICE" in
    *"Tiếng Việt"*) LANGUAGE="vi" ;;
    *"English"*)    LANGUAGE="en" ;;
    *"中文"*)        LANGUAGE="zh" ;;
    *)              LANGUAGE="en" ;;
esac

# ─────────────────────────────────────────────
# ROOT CHECK
# ─────────────────────────────────────────────

if [ "$(id -u)" -eq 0 ]; then
    SUDO=""
else
    if ! command -v sudo >/dev/null 2>&1; then
        log_error "$(tr need_root)"
        log_info "$(tr run_with_sudo)"
        echo ""
        log_dim "sudo bash <(curl -Ls https://raw.githubusercontent.com/ngthanhvu/docker-manager/refs/heads/main/install.sh)"
        echo ""
        exit 1
    fi
    SUDO="sudo"
fi

# ─────────────────────────────────────────────
# STEP 2 — Action
# ─────────────────────────────────────────────

show_banner
show_step 2 3 "$(tr step_action)"

arrow_select "$(tr step_action)" ACTION_CHOICE \
    "$(tr menu_install)" \
    "$(tr menu_uninstall)"

# ─────────────────────────────────────────────
# HELPERS
# ─────────────────────────────────────────────

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

    [ -z "$latest_tag" ] && latest_tag="latest"
    echo "$latest_tag"
}

set_image_refs() {
    local resolved_tag="${IMAGE_TAG:-}"
    if [ -z "$resolved_tag" ]; then
        log_info "$(tr resolving_latest)"
        resolved_tag="$(resolve_latest_tag "${REPO_PREFIX}-frontend")"
    fi

    BACKEND_IMAGE="${DOCKERHUB_NAMESPACE}/${REPO_PREFIX}-backend:${resolved_tag}"
    FRONTEND_IMAGE="${DOCKERHUB_NAMESPACE}/${REPO_PREFIX}-frontend:${resolved_tag}"

    log_success "$(tr using_image_tag) ${BOLD}${resolved_tag}${RESET}"
    log_dim "$(tr backend_image)  ${BACKEND_IMAGE}"
    log_dim "$(tr frontend_image) ${FRONTEND_IMAGE}"
    echo ""
}

# ─────────────────────────────────────────────
# INSTALL
# ─────────────────────────────────────────────

install_app() {
    show_banner
    show_step 3 3 "$(tr step_config)"

    # Check Docker
    if ! command -v docker &>/dev/null; then
        log_warn "$(tr docker_not_found)"
        curl -fsSL https://get.docker.com | $SUDO sh
        echo ""
    fi

    # Check Compose
    if ! docker compose version &>/dev/null; then
        log_error "$(tr compose_required)"
        exit 1
    fi

    set_image_refs

    # Port input
    printf "  ${BOLD}${WHITE}$(tr enter_port)${RESET} ${DIM}[${DEFAULT_PORT}]${RESET}  "
    read -r PORT
    PORT=${PORT:-$DEFAULT_PORT}

    check_port() {
        if lsof -i:"$1" >/dev/null 2>&1; then return 1; else return 0; fi
    }

    while ! check_port "$PORT"; do
        printf "\n"
        log_warn "$(printf "$(tr port_in_use)" "$PORT")"
        printf "  ${YELLOW}$(tr choose_another_port)${RESET}"
        read -r PORT
    done

    log_success "$(tr using_port) ${BOLD}${PORT}${RESET}"
    echo ""

    # ── Install ──────────────────────────────
    show_banner
    show_step 3 3 "$(tr step_install)"

    log_info "$(tr installing)"
    echo ""

    $SUDO mkdir -p "$APP_DIR"
    cd "$APP_DIR"

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

    log_info "$(tr pulling_images)"
    $SUDO docker compose -f "$APP_DIR/docker-compose.yml" pull
    echo ""

    log_info "$(tr starting_app)"
    $SUDO docker compose -f "$APP_DIR/docker-compose.yml" up -d
    echo ""

    IP=$(hostname -I | awk '{print $1}')

    # ── Success box ───────────────────────────
    show_banner
    printf "  ${GREEN}${BOLD}✔  $(tr installed_title)${RESET}\n\n"
    printf "  ${DIM}$(printf '─%.0s' {1..44})${RESET}\n"
    printf "\n"
    printf "  ${DIM}$(tr access_url)${RESET}\n"
    printf "  ${CYAN}${BOLD}http://${IP}:${PORT}${RESET}\n"
    printf "\n"
    printf "  ${DIM}$(tr install_directory)${RESET}\n"
    printf "  ${WHITE}${APP_DIR}${RESET}\n"
    printf "\n"
    printf "  ${DIM}$(printf '─%.0s' {1..44})${RESET}\n"
    echo ""
}

# ─────────────────────────────────────────────
# UNINSTALL
# ─────────────────────────────────────────────

uninstall_app() {
    show_banner

    if [ ! -d "$APP_DIR" ]; then
        log_warn "$(tr not_installed)"
        echo ""
        exit 0
    fi

    printf "  ${YELLOW}$(tr confirm_uninstall)${RESET}"
    read -r CONFIRM

    if [ "$CONFIRM" != "y" ]; then
        echo ""
        log_info "$(tr cancelled)"
        echo ""
        exit 0
    fi

    echo ""
    cd "$APP_DIR"

    INSTALLED_BACKEND_IMAGE=$(sed -n '/backend:/,/frontend:/ s/^[[:space:]]*image:[[:space:]]*//p' "$APP_DIR/docker-compose.yml" | head -n 1)
    INSTALLED_FRONTEND_IMAGE=$(sed -n '/frontend:/,$ s/^[[:space:]]*image:[[:space:]]*//p' "$APP_DIR/docker-compose.yml" | head -n 1)

    log_info "$(tr stopping_containers)"
    $SUDO docker compose -f "$APP_DIR/docker-compose.yml" down

    log_info "$(tr removing_images)"
    $SUDO docker image rm "${INSTALLED_BACKEND_IMAGE:-$BACKEND_IMAGE}" 2>/dev/null || true
    $SUDO docker image rm "${INSTALLED_FRONTEND_IMAGE:-$FRONTEND_IMAGE}" 2>/dev/null || true

    log_info "$(tr removing_install_dir)"
    $SUDO rm -rf "$APP_DIR"

    echo ""
    log_success "$(tr removed_success)"
    echo ""
}

# ─────────────────────────────────────────────
# DISPATCH
# ─────────────────────────────────────────────

case "$ACTION_CHOICE" in
    *"$(tr menu_install)"*)    install_app ;;
    *"$(tr menu_uninstall)"*)  uninstall_app ;;
    *)
        log_error "$(tr invalid_option)"
        ;;
esac