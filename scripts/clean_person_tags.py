#!/usr/bin/env python3
"""清理没有人脸的照片的人物标签"""
import sqlite3, sys

db_path = sys.argv[1] if len(sys.argv) > 1 else "data/photos.db"
person_tags = {"亲子时光", "人物合影", "单人肖像", "儿童", "老人", "家庭合照", "人", "人群聚集"}

conn = sqlite3.connect(db_path)
# 找出没有人脸的照片ID
no_face = conn.execute("SELECT id FROM photos WHERE has_faces=0").fetchall()
removed = 0
for (pid,) in no_face:
    conn.execute("DELETE FROM tags WHERE photo_id=? AND tag IN ({})".format(
        ",".join(f"'{t}'" for t in person_tags)), (pid,))
    removed += conn.total_changes

conn.commit()
print(f"清理完成: 从 {len(no_face)} 张无人脸照片中移除 {removed} 个人物标签")
conn.close()
