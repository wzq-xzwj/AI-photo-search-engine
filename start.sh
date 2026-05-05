#!/bin/bash
# PhotoLens AI - 一键启动脚本
# 启动所有服务：ML Service (InsightFace + CLIP)、Go Backend、Frontend

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# 加载环境变量
if [ -f "$PROJECT_DIR/.env" ]; then
    export $(grep -v '^#' "$PROJECT_DIR/.env" | xargs)
    echo -e "${GREEN}  ✓ 已加载 .env${NC}"
elif [ -f "$PROJECT_DIR/.env.deepseek" ]; then
    export $(grep -v '^#' "$PROJECT_DIR/.env.deepseek" | xargs)
    echo -e "${GREEN}  ✓ 已加载 .env.deepseek${NC}"
else
    echo -e "${YELLOW}  ⚠ 未找到 .env 文件，使用默认配置${NC}"
fi

# 启动 ML Service
echo -e "${YELLOW}[2/5] 启动 ML Service (端口 8000)...${NC}"
cd "$PROJECT_DIR/ml-service"
if [ ! -d "venv" ]; then
    echo -e "${YELLOW}  创建虚拟环境...${NC}"
    python3 -m venv venv
    source venv/bin/activate
    pip install -r requirements.txt
else
    source venv/bin/activate
fi
nohup python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 > /tmp/ml-service.log 2>&1 &
ML_PID=$!
echo "  ML Service PID: $ML_PID"

# 等待 ML Service 就绪
echo -e "${YELLOW}[3/5] 等待 ML Service 就绪...${NC}"
for i in {1..120}; do
    if curl -sf http://127.0.0.1:8000/api/v1/health >/dev/null 2>&1; then
        echo -e "${GREEN}  ✓ ML Service 已就绪${NC}"
        break
    fi
    if ! kill -0 $ML_PID 2>/dev/null; then
        echo -e "${RED}  ✗ ML Service 启动失败，查看日志: /tmp/ml-service.log${NC}"
        exit 1
    fi
    echo -n "."
    sleep 1
done
echo ""

# 编译 Go Backend (如果需要)
echo -e "${YELLOW}[4/5] 启动 Go Backend (端口 8080)...${NC}"
cd "$PROJECT_DIR"
if [ ! -f "photo-server" ] || [ "cmd/server/main.go" -nt "photo-server" ]; then
    echo "  编译 Go Backend..."
    go build -o photo-server ./cmd/server/main.go
fi
nohup ./photo-server > /tmp/go-backend.log 2>&1 &
GO_PID=$!
echo "  Go Backend PID: $GO_PID"

sleep 2
if curl -sf http://127.0.0.1:8080/api/v1/dirs >/dev/null 2>&1; then
    echo -e "${GREEN}  ✓ Go Backend 已就绪${NC}"
else
    echo -e "${RED}  ✗ Go Backend 启动失败，查看日志: /tmp/go-backend.log${NC}"
    exit 1
fi

# 启动 Frontend
echo -e "${YELLOW}[5/5] 启动 Frontend (端口 5173)...${NC}"
cd "$PROJECT_DIR/web"
if [ ! -d "node_modules" ]; then
    echo "  安装前端依赖..."
    npm install
fi
nohup npm run dev > /tmp/frontend.log 2>&1 &
FE_PID=$!
echo "  Frontend PID: $FE_PID"

sleep 3
FRONTEND_PORT=$(grep -o 'localhost:[0-9]*' /tmp/frontend.log | head -1 | cut -d: -f2)
if [ -z "$FRONTEND_PORT" ]; then
    FRONTEND_PORT=5173
fi
if curl -sf "http://127.0.0.1:$FRONTEND_PORT" >/dev/null 2>&1; then
    echo -e "${GREEN}  ✓ Frontend 已就绪${NC}"
else
    echo -e "${YELLOW}  ⚠ Frontend 可能需要更长时间启动${NC}"
fi

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  所有服务已启动！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "访问地址:"
echo -e "  ${BLUE}前端界面:${NC} http://localhost:$FRONTEND_PORT"
echo -e "  ${BLUE}后端 API:${NC} http://localhost:8080"
echo -e "  ${BLUE}ML 服务:${NC}  http://localhost:8000"
echo ""
echo -e "日志文件:"
echo -e "  ${YELLOW}ML Service:${NC}  /tmp/ml-service.log"
echo -e "  ${YELLOW}Go Backend:${NC}   /tmp/go-backend.log"
echo -e "  ${YELLOW}Frontend:${NC}     /tmp/frontend.log"
echo ""
echo -e "停止服务: ${RED}./stop.sh${NC}"
echo ""

# 保存 PID
echo "$ML_PID" > /tmp/photolens.pids
echo "$GO_PID" >> /tmp/photolens.pids
echo "$FE_PID" >> /tmp/photolens.pids
