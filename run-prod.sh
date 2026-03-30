#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROD_COMPOSE_FILE="$ROOT_DIR/docker-compose.prod.yml"

LOCAL_BACKEND_IMAGE="docker-manager-backend:prod-build"
LOCAL_FRONTEND_IMAGE="docker-manager-frontend:prod-build"

usage() {
  cat <<USAGE
Usage:
  $(basename "$0") [command] [options]

Commands:
  up [service]         Start prod stack (default service: all)
  down                 Stop prod stack
  restart [service]    Restart prod stack or 1 service
  logs [service]       Follow logs
  build [service]      Build prod images from compose
  push <user> <repo_prefix> [service] [tag]
                       Build + tag + push to Docker Hub

Service:
  backend | frontend | all (default: all)

Examples:
  $(basename "$0")
  $(basename "$0") up
  $(basename "$0") up backend
  $(basename "$0") logs frontend
  $(basename "$0") push yourname docker-manager all v1.0.0
USAGE
}

require_docker() {
  if ! command -v docker >/dev/null 2>&1; then
    echo "Error: chưa có lệnh docker trong PATH"
    exit 1
  fi

  if ! docker info >/dev/null 2>&1; then
    echo "Error: Docker daemon chưa chạy hoặc không truy cập được."
    exit 1
  fi
}

require_compose_file() {
  if [[ ! -f "$PROD_COMPOSE_FILE" ]]; then
    echo "Error: không tìm thấy file docker-compose.prod.yml"
    exit 1
  fi
}

normalize_service() {
  local svc="${1:-all}"
  case "$svc" in
    backend|frontend|all) echo "$svc" ;;
    *)
      echo "Error: service phải là backend | frontend | all"
      exit 1
      ;;
  esac
}

compose_build() {
  local svc
  svc="$(normalize_service "${1:-all}")"
  local app_version="${2:-prod-build}"
  local build_date="${3:-$(date -u +%F)}"

  if [[ "$svc" == "all" ]]; then
    APP_VERSION="$app_version" BUILD_DATE="$build_date" docker compose -f "$PROD_COMPOSE_FILE" build backend frontend
  else
    APP_VERSION="$app_version" BUILD_DATE="$build_date" docker compose -f "$PROD_COMPOSE_FILE" build "$svc"
  fi
}

tag_and_push() {
  local svc="$1"
  local user="$2"
  local repo_prefix="$3"
  local tag="$4"
  local local_image remote_image

  if [[ "$svc" == "backend" ]]; then
    local_image="$LOCAL_BACKEND_IMAGE"
    remote_image="${user}/${repo_prefix}-backend:${tag}"
  else
    local_image="$LOCAL_FRONTEND_IMAGE"
    remote_image="${user}/${repo_prefix}-frontend:${tag}"
  fi

  echo "==> Tagging $local_image -> $remote_image"
  docker tag "$local_image" "$remote_image"

  echo "==> Pushing $remote_image"
  docker push "$remote_image"
}

push_images() {
  local user="$1"
  local repo_prefix="$2"
  local svc
  svc="$(normalize_service "${3:-all}")"
  local tag="${4:-latest}"

  echo "==> Building prod images từ docker-compose.prod.yml"
  compose_build "$svc" "$tag" "$(date -u +%F)"

  if [[ "$svc" == "all" ]]; then
    tag_and_push backend "$user" "$repo_prefix" "$tag"
    tag_and_push frontend "$user" "$repo_prefix" "$tag"
  else
    tag_and_push "$svc" "$user" "$repo_prefix" "$tag"
  fi

  echo "Hoàn tất build & push."
}

main() {
  cd "$ROOT_DIR"

  local cmd="${1:-up}"
  shift || true

  case "$cmd" in
    -h|--help|help)
      usage
      ;;
    up)
      require_compose_file
      require_docker
      local svc
      svc="$(normalize_service "${1:-all}")"
      if [[ "$svc" == "all" ]]; then
        APP_VERSION="${APP_VERSION:-prod-build}" BUILD_DATE="${BUILD_DATE:-$(date -u +%F)}" docker compose -f "$PROD_COMPOSE_FILE" up --build -d
      else
        APP_VERSION="${APP_VERSION:-prod-build}" BUILD_DATE="${BUILD_DATE:-$(date -u +%F)}" docker compose -f "$PROD_COMPOSE_FILE" up --build -d "$svc"
      fi
      ;;
    down)
      require_compose_file
      require_docker
      docker compose -f "$PROD_COMPOSE_FILE" down
      ;;
    restart)
      require_compose_file
      require_docker
      local svc
      svc="$(normalize_service "${1:-all}")"
      if [[ "$svc" == "all" ]]; then
        docker compose -f "$PROD_COMPOSE_FILE" restart
      else
        docker compose -f "$PROD_COMPOSE_FILE" restart "$svc"
      fi
      ;;
    logs)
      require_compose_file
      require_docker
      local svc
      svc="$(normalize_service "${1:-all}")"
      if [[ "$svc" == "all" ]]; then
        docker compose -f "$PROD_COMPOSE_FILE" logs -f
      else
        docker compose -f "$PROD_COMPOSE_FILE" logs -f "$svc"
      fi
      ;;
    build)
      require_compose_file
      require_docker
      compose_build "${1:-all}" "${APP_VERSION:-prod-build}" "${BUILD_DATE:-$(date -u +%F)}"
      ;;
    push)
      require_compose_file
      require_docker
      if [[ $# -lt 2 ]]; then
        echo "Error: thiếu tham số cho lệnh push"
        echo "Ví dụ: $(basename "$0") push yourname docker-manager all v1.0.0"
        exit 1
      fi
      push_images "$1" "$2" "${3:-all}" "${4:-latest}"
      ;;
    *)
      echo "Error: command không hợp lệ: $cmd"
      usage
      exit 1
      ;;
  esac
}

main "$@"
