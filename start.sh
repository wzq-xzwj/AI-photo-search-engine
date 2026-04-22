#!/bin/bash
# PhotoLens AI - 一键启动脚本
# 启动所有服务：ML Service、Go Backend、Frontend

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目目录
PROJECT_DIR="/Users/wuzhaoqing/Pictures/photo-search-engine"
cd "$PROJECT_DIR"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}  PhotoLens AI - 智能照片搜索引擎${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 检查并关闭已运行的服务
echo -e "${YELLOW}[1/5] 检查并关闭已运行的服务...${NC}"
pkill -f "uvicorn app.main:app" 2>/dev/null || true
pkill -f "photo-server" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true
sleep 2

# 启动 ML Service
echo -e "${YELLOW}[2/5] 启动 ML Service (端口 8000)...${NC}"
cd "$PROJECT_DIR/ml-service"
source venv/bin/activate
nohup python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 > /tmp/ml-service.log 2>&1 &
ML_PID=$!
echo "ML Service PID: $ML_PID"

# 等待 ML Service 就绪
echo -e "${YELLOW}[3/5] 等待 ML Service 就绪...${NC}"
for i in {1..60}; do
    if curl -sf http://127.0.0.1:8000/api/v1/health >/dev/null 2>&1; then
        echo -e "${GREEN}✓ ML Service 已就绪${NC}"
        break
    fi
    if ! kill -0 $ML_PID 2>/dev/null; then
        echo -e "${RED}✗ ML Service 启动失败${NC}"
        exit 1
    fi
    echo -n "."
    sleep 1
done
echo ""

# 启动 Go Backend
echo -e "${YELLOW}[4/5] 启动 Go Backend (端口 8080)...${NC}"
cd "$PROJECT_DIR"
export LLM_PROVIDER=openai
export OPENAI_BASE_URL=https://api.deepseek.com/v1
export OPENAI_API_KEY=sk-13d33f25229248d28915daf3761ff0e7
export OPENAI_MODEL=deepseek-chat

nohup ./photo-server > /tmp/go-backend.log 2>&1 &
GO_PID=$!
echo "Go Backend PID: $GO_PID"

# 等待 Go Backend 就绪
sleep 2
if curl -sf http://127.0.0.1:8080/api/v1/health >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Go Backend 已就绪${NC}"
else
    echo -e "${RED}✗ Go Backend 启动失败，查看日志: /tmp/go-backend.log${NC}"
    exit 1
fi

# 启动 Frontend
echo -e "${YELLOW}[5/5] 启动 Frontend (端口 5173)...${NC}"
cd "$PROJECT_DIR/web"
nohup npm run dev > /tmp/frontend.log 2>&1 &
FE_PID=$!
echo "Frontend PID: $FE_PID"

# 等待 Frontend 就绪
sleep 3
if curl -sf http://127.0.0.1:5173 >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Frontend 已就绪${NC}"
else
    echo -e "${YELLOW}⚠ Frontend 可能需要更长时间启动${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  所有服务已启动！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "访问地址:"
echo -e "  ${BLUE}前端界面:${NC} http://localhost:5173"
echo -e "  ${BLUE}后端 API:${NC} http://localhost:8080"
echo -e "  ${BLUE}ML 服务:${NC}  http://localhost:8000"
echo ""
echo -e "日志文件:"
echo -e "  ${YELLOW}ML Service:${NC}  /tmp/ml-service.log"
echo -e "  ${YELLOW}Go Backend:${NC}   /tmp/go-backend.log"
echo -e "  ${Yellow}Frontend:${NC}     /tmp/frontend.log"
echo ""
echo -e "停止服务: ${RED}./stop.sh${NC}"
echo ""

# 保存 PID 到文件
echo "$ML_PID" > /tmp/photolens.pids
echo "$GO_PID" >> /tmp/photolens.pids
echo "$FE_PID" >> /tmp/photolens.pids
