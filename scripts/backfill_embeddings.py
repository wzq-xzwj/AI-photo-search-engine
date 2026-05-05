#!/usr/bin/env python3
"""为缺少 embedding 的照片回填 CLIP embedding"""
import sqlite3, requests, struct, math, sys, os
from pathlib import Path

DB_PATH = sys.argv[1] if len(sys.argv) > 1 else "data/photos.db"
ML_URL = os.environ.get("ML_SERVICE_URL", "http://127.0.0.1:8000")

conn = sqlite3.connect(DB_PATH)

# 确保 embedding 列存在
try:
    conn.execute("ALTER TABLE photos ADD COLUMN embedding BLOB")
except:
    pass

# 找缺少 embedding 的照片
rows = conn.execute("SELECT id, path FROM photos WHERE embedding IS NULL OR length(embedding)=0").fetchall()
if not rows:
    print("所有照片已有 embedding")
    conn.close()
    sys.exit(0)

print(f"需回填: {len(rows)} 张")

from PIL import Image
import io

def get_embedding(image_path):
    """调用 ML 服务提取 embedding"""
    if not os.path.exists(image_path):
        return None
    with open(image_path, 'rb') as f:
        resp = requests.post(f"{ML_URL}/api/v1/extract", files={"file": f})
    if resp.status_code != 200:
        return None
    return resp.json()["embedding"]

def floats_to_bytes(v):
    return b''.join(struct.pack('<f', x) for x in v)

count = 0
for i, (pid, path) in enumerate(rows):
    emb = get_embedding(path)
    if emb is None:
        continue
    data = floats_to_bytes(emb)
    conn.execute("UPDATE photos SET embedding=?, updated_at=CURRENT_TIMESTAMP WHERE id=?", (data, pid))
    count += 1
    if (i+1) % 50 == 0 or i == len(rows)-1:
        conn.commit()
        print(f"进度: {i+1}/{len(rows)} ({count} 成功)")
    if (i+1) % 10 == 0:
        conn.commit()

conn.commit()
conn.close()
print(f"回填完成: {count}/{len(rows)} 张")
