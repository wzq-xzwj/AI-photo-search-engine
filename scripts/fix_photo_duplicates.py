#!/usr/bin/env python3
"""
修复照片内重复人脸问题
同一照片内的多个人脸不允许归为同一人
策略：保留置信度最高的，其他标记为Unknown
"""

import sqlite3
import os

DB_PATH = os.path.expanduser("~/Pictures/photo-search-engine/data/photos.db")

def fix_photo_duplicates():
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()
    
    # 找到所有有重复的照片-人物对
    cursor.execute("""
        SELECT person_id, photo_id, COUNT(*) as cnt
        FROM faces
        WHERE person_id IS NOT NULL
        GROUP BY person_id, photo_id
        HAVING cnt > 1
    """)
    
    duplicates = cursor.fetchall()
    
    if not duplicates:
        print("✅ 没有发现照片内重复")
        return
    
    print(f"🚨 发现 {len(duplicates)} 个照片内重复:")
    
    total_fixed = 0
    
    for person_id, photo_id, count in duplicates:
        print(f"\n   {person_id} + {photo_id}: {count} 个脸")
        
        # 获取这些人脸，按置信度排序
        cursor.execute("""
            SELECT id, confidence
            FROM faces
            WHERE person_id = ? AND photo_id = ?
            ORDER BY confidence DESC
        """, (person_id, photo_id))
        
        faces = cursor.fetchall()
        
        # 保留第一个（置信度最高），其他标记为Unknown
        keep_id = faces[0][0]
        remove_ids = [f[0] for f in faces[1:]]
        
        print(f"      保留: face_id={keep_id}, confidence={faces[0][1]:.2f}")
        
        for rid in remove_ids:
            cursor.execute("""
                UPDATE faces
                SET person_id = NULL, is_unknown = TRUE
                WHERE id = ?
            """, (rid,))
            total_fixed += 1
            
        print(f"      标记Unknown: {len(remove_ids)} 个")
    
    conn.commit()
    conn.close()
    
    print(f"\n✅ 修复完成: {total_fixed} 个重复人脸已标记为Unknown")

if __name__ == "__main__":
    fix_photo_duplicates()
