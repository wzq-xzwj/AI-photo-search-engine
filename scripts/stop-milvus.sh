#!/bin/bash

set -e

echo "🛑 停止 Milvus..."

cd "$(dirname "$0")/../deploy"

docker-compose down

echo "✅ Milvus 已停止"