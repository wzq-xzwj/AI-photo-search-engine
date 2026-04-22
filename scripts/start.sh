#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

cleanup() {
  echo ""
  echo "正在关闭服务..."
  kill $PID_ML $PID_GO $PID_WEB 2>/dev/null || true
  wait 2>/dev/null || true
  echo "服务已关闭"
  exit
}
trap cleanup INT TERM

echo "启动 ML Service (http://127.0.0.1:8000)..."
cd ml-service
source venv/bin/activate
python -m uvicorn app.main:app --host 0.0.0.0 --port 8000 &
PID_ML=$!
cd ..

echo "等待 ML Service 就绪..."
for i in {1..60}; do
  if curl -sf http://127.0.0.1:8000/api/v1/health >/dev/null 2>&1; then
    echo "✅ ML Service 已就绪"
    break
  fi
  if ! kill -0 $PID_ML 2>/dev/null; then
    echo "❌ ML Service 启动失败"
    exit 1
  fi
  sleep 1
done

echo "启动 Go 后端 (http://127.0.0.1:8080)..."
go run cmd/server/main.go &
PID_GO=$!

echo "启动前端开发服务器..."
cd web
if [ ! -d "node_modules" ]; then
  echo "正在安装前端依赖..."
  npm install
fi
npm run dev &
PID_WEB=$!
cd ..

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  全部服务已启动"
echo "  ML 服务:  http://127.0.0.1:8000"
echo "  Go 后端:  http://127.0.0.1:8080"
echo "  前端:     见上方 Vite 输出"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  按 Ctrl+C 关闭所有服务"
echo ""

wait
