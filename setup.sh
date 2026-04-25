#!/usr/bin/env bash
# ============================================
# PhotoLens AI - 一键部署脚本
# 用法: bash setup.sh
# ============================================
set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$PROJECT_DIR"

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; }
info() { echo -e "${CYAN}[→]${NC} $1"; }

cleanup() {
    echo ""
    warn "正在关闭服务..."
    kill $PID_ML $PID_GO $PID_WEB 2>/dev/null || true
    wait 2>/dev/null || true
    log "服务已关闭"
    exit
}
trap cleanup INT TERM

echo ""
echo -e "${CYAN}╔══════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║     PhotoLens AI - 一键部署脚本         ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════╝${NC}"
echo ""

# ============================================
# Step 1: 检查系统依赖
# ============================================
info "检查系统依赖..."

check_cmd() {
    if command -v "$1" &>/dev/null; then
        log "$1 $( $1 --version 2>/dev/null | head -1 || echo 'OK' )"
    else
        err "$1 未安装，请先安装"
        exit 1
    fi
}

check_cmd go
check_cmd python3
check_cmd node
check_cmd npm

# 检查 Python 版本
PY_VER=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
if [[ "$(echo "$PY_VER >= 3.10" | bc 2>/dev/null || echo 0)" != "1" ]]; then
    warn "Python 版本: $PY_VER (建议 >= 3.10)"
fi

log "系统依赖检查通过"
echo ""

# ============================================
# Step 2: 配置环境变量
# ============================================
info "配置环境变量..."

if [ ! -f .env ]; then
    if [ -f .env.example ]; then
        cp .env.example .env
        warn "已创建 .env 文件，请编辑填入你的 API Key:"
        echo "     vim .env"
        echo ""
        warn "⚠️  如果跳过此步骤，LLM 自然语言解析功能将不可用"
        echo ""
    else
        warn ".env.example 不存在，跳过"
    fi
else
    log ".env 已存在"
fi
echo ""

# ============================================
# Step 3: 安装 Go 依赖
# ============================================
info "安装 Go 依赖..."
go mod download 2>/dev/null && log "Go 依赖安装完成" || warn "Go 依赖安装失败（可能是网络问题）"
echo ""

# ============================================
# Step 4: 安装 Python ML 服务
# ============================================
info "设置 Python ML 服务..."

cd "$PROJECT_DIR/ml-service"

if [ ! -d "venv" ]; then
    python3 -m venv venv
    log "创建虚拟环境"
fi

source venv/bin/activate
pip install -q --upgrade pip 2>/dev/null
pip install -q -r requirements.txt 2>/dev/null && log "Python 依赖安装完成" || warn "Python 依赖安装失败"
deactivate
cd "$PROJECT_DIR"
echo ""

# ============================================
# Step 5: 安装前端依赖
# ============================================
info "安装前端依赖..."
cd "$PROJECT_DIR/web"
if [ ! -d "node_modules" ]; then
    npm install --silent 2>/dev/null && log "前端依赖安装完成" || warn "前端依赖安装失败，尝试手动: cd web && npm install"
else
    log "前端依赖已安装"
fi
cd "$PROJECT_DIR"
echo ""

# ============================================
# Step 6: 启动服务
# ============================================
info "启动服务..."

# ML Service
cd "$PROJECT_DIR/ml-service"
source venv/bin/activate
python3 -m uvicorn app.main:app --host 0.0.0.0 --port 8000 &
PID_ML=$!
deactivate
cd "$PROJECT_DIR"

info "等待 ML 服务就绪..."
for i in $(seq 1 60); do
    if curl -sf http://127.0.0.1:8000/api/v1/health >/dev/null 2>&1; then
        log "ML 服务已就绪 (http://127.0.0.1:8000)"
        break
    fi
    if ! kill -0 $PID_ML 2>/dev/null; then
        err "ML 服务启动失败，请检查 ml-service/ 日志"
        exit 1
    fi
    sleep 1
done

# Go Backend
cd "$PROJECT_DIR"
go run cmd/server/main.go &
PID_GO=$!
sleep 2
if curl -sf http://127.0.0.1:8080/api/v1/dirs >/dev/null 2>&1; then
    log "Go 后端已就绪 (http://127.0.0.1:8080)"
else
    warn "Go 后端可能还在启动中..."
fi

# Vite Frontend
cd "$PROJECT_DIR/web"
npm run dev &
PID_WEB=$!
cd "$PROJECT_DIR"

sleep 3
echo ""

# ============================================
# 部署完成
# ============================================
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}  ✅ PhotoLens AI 部署完成！${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "  ${CYAN}ML 服务:${NC}  http://127.0.0.1:8000"
echo -e "  ${CYAN}Go 后端:${NC}  http://127.0.0.1:8080"
echo -e "  ${CYAN}前端页面:${NC} http://localhost:5173"
echo ""
echo -e "  ${YELLOW}按 Ctrl+C 关闭所有服务${NC}"
echo ""

wait
