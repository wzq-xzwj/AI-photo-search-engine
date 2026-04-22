#!/bin/bash

set -e

echo "🚀 启动 Milvus..."

# 检查 Docker
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker 未启动，请先启动 Docker Desktop"
    exit 1
fi

# 进入部署目录
cd "$(dirname "$0")/../deploy"

# 启动服务
echo "📦 启动 Milvus 服务..."
docker-compose up -d

# 等待 Milvus 就绪
echo "⏳ 等待 Milvus 就绪..."
for i in {1..60}; do
    if curl -s http://localhost:9091/healthz > /dev/null 2>&1; then
        echo ""
        echo "✅ Milvus 已就绪！"
        echo ""
        echo "服务地址："
        echo "  - Milvus gRPC:  localhost:19530"
        echo "  - Milvus HTTP:  http://localhost:9091"
        echo "  - Attu 管理界面: http://localhost:8000"
        echo "  - MinIO 控制台:  http://localhost:9001"
        echo ""
        echo "启动照片搜索引擎："
        echo "  cd .. && ./start.sh"
        exit 0
    fi
    echo -n "."
    sleep 1
done

echo ""
echo "❌ Milvus 启动超时"
echo "查看日志：docker-compose logs -f milvus-standalone"
exit 1