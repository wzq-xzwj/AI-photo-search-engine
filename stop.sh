#!/bin/bash
# PhotoLens AI - 停止脚本

echo "正在停止 PhotoLens AI 服务..."

# 从 PID 文件读取并停止
if [ -f /tmp/photolens.pids ]; then
    while read pid; do
        if kill -0 "$pid" 2>/dev/null; then
            echo "停止进程 $pid"
            kill "$pid" 2>/dev/null || true
        fi
    done < /tmp/photolens.pids
    rm /tmp/photolens.pids
fi

# 额外确保所有相关进程被停止
pkill -f "uvicorn app.main:app" 2>/dev/null || true
pkill -f "photo-server" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true

echo "所有服务已停止"
