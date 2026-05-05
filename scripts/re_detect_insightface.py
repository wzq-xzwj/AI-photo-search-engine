#!/usr/bin/env python3
"""
使用 InsightFace 重新检测所有照片的人脸
并用 DBSCAN 聚类，替换 face_recognition 的结果
"""

import sys
import os
import json
import time
import sqlite3
import cv2
import numpy as np
from pathlib import Path
from sklearn.cluster import DBSCAN
from sklearn.preprocessing import normalize

# Add ml-service to path for imports
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), '..', 'ml-service'))
from insightface_service import detect_faces_insightface, extract_thumbnail

DB_PATH = '/Users/wuzhaoqing/Pictures/photo-search-engine/data/photos.db'
THUMBS_DIR = '/Users/wuzhaoqing/Pictures/photo-search-engine/data/faces'


def main():
    os.makedirs(THUMBS_DIR, exist_ok=True)
    conn = sqlite3.connect(DB_PATH)

    # 1. 清理旧的人脸数据
    conn.execute('DELETE FROM faces')
    conn.execute('DELETE FROM persons')
    conn.commit()
    print("✅ 清理旧数据完成")

    # 2. 获取所有照片
    photos = conn.execute('SELECT id, path FROM photos').fetchall()
    total = len(photos)
    print(f"📸 共 {total} 张照片，开始检测...")

    all_faces = []  # (face_row, encoding_list) for clustering
    face_count = 0
    photos_with_faces = 0
    failed = 0
    start_time = time.time()

    for i, (photo_id, photo_path) in enumerate(photos):
        if not os.path.exists(photo_path):
            failed += 1
            continue

        ext = os.path.splitext(photo_path)[1].lower()
        if ext not in ('.jpg', '.jpeg', '.png', '.tif', '.tiff', '.bmp', '.webp', '.heic', '.dng'):
            continue

        try:
            faces = detect_faces_insightface(
                photo_path,
                min_det_score=0.5,
                min_size=60,
            )

            if not faces:
                continue

            photos_with_faces += 1

            for face in faces:
                face_index = face['index']
                location_json = json.dumps(face['location'])
                encoding_json = json.dumps(face['encoding'])

                # 提取缩略图
                thumbnail_path = os.path.join(
                    THUMBS_DIR, f"if_{photo_id}_{face_index}.jpg"
                )
                thumbnail_data = extract_thumbnail(
                    photo_path, face['location'], thumbnail_path
                )
                if thumbnail_data is None:
                    thumbnail_path = ''

                # 保存人脸记录
                conn.execute(
                    """INSERT INTO faces 
                    (photo_id, face_index, person_id, location, encoding, 
                     confidence, thumbnail_path, recognition_confidence, is_unknown)
                    VALUES (?, ?, NULL, ?, ?, ?, ?, ?, 0)""",
                    (
                        photo_id, face_index,
                        location_json, encoding_json,
                        face['confidence'], thumbnail_path,
                        face['det_score'],  # recognition_confidence = InsightFace det_score
                    )
                )
                face_count += 1

                # 保存编码用于聚类
                all_faces.append((
                    conn.execute('SELECT last_insert_rowid()').fetchone()[0],
                    np.array(face['encoding'], dtype=np.float64),
                ))

        except Exception as e:
            failed += 1
            if failed <= 5:
                print(f"  ⚠️ {os.path.basename(photo_path)}: {e}")

        # 进度报告
        elapsed = time.time() - start_time
        if (i + 1) % 50 == 0:
            eta = (elapsed / (i + 1)) * (total - i - 1) if i > 0 else 0
            print(
                f"  进度: {i+1}/{total} "
                f"({(i+1)/total*100:.0f}%) "
                f"| 人脸: {face_count} "
                f"| 耗时: {elapsed:.0f}s "
                f"| ETA: {eta:.0f}s"
            )

    conn.commit()
    elapsed = time.time() - start_time
    print(f"\n✅ 检测完成！")
    print(f"  耗时: {elapsed:.0f}s")
    print(f"  照片含人脸: {photos_with_faces}/{total}")
    print(f"  人脸总数: {face_count}")
    print(f"  失败: {failed}")

    # 3. DBSCAN 聚类
    if len(all_faces) < 2:
        print("⚠️ 人脸数不足，跳过聚类")
        conn.close()
        return

    print(f"\n🔬 开始 DBSCAN 聚类 (共 {len(all_faces)} 张人脸)...")

    face_ids = [f[0] for f in all_faces]
    encodings = np.array([f[1] for f in all_faces], dtype=np.float64)

    # InsightFace 的 embedding 已经是 L2 归一化的
    # DBSCAN 参数：
    # - eps: 最大距离，InsightFace 相同人脸距离通常 < 0.4，不同人脸 > 0.6
    # - min_samples: 最少组成簇的人脸数
    clusterer = DBSCAN(eps=0.35, min_samples=2, metric='cosine')
    labels = clusterer.fit_predict(encodings)

    n_clusters = len(set(labels)) - (1 if -1 in labels else 0)
    n_noise = list(labels).count(-1)
    print(f"  聚类结果: {n_clusters} 个人物, {n_noise} 噪声点")

    # 保存聚类结果
    for fid, label in zip(face_ids, labels):
        if label == -1:
            # 噪声点标记为 unknown
            conn.execute(
                'UPDATE faces SET person_id = NULL, is_unknown = 1 WHERE id = ?',
                (fid,)
            )
        else:
            person_id = f"person_{label}"
            conn.execute(
                'UPDATE faces SET person_id = ?, is_unknown = 0 WHERE id = ?',
                (person_id, fid)
            )

    # 4. 同步 persons 表
    conn.execute('DELETE FROM persons')
    conn.execute("""
        INSERT INTO persons (id, name, avatar, face_count, photo_count, created_at)
        SELECT 
            person_id,
            person_id as name,
            (SELECT thumbnail_path FROM faces f2 
             WHERE f2.person_id = f1.person_id 
               AND f2.thumbnail_path != '' 
             ORDER BY f2.confidence DESC 
             LIMIT 1
            ) as avatar,
            COUNT(*) as face_count,
            COUNT(DISTINCT photo_id) as photo_count,
            CURRENT_TIMESTAMP
        FROM faces f1
        WHERE person_id IS NOT NULL AND person_id != ''
        GROUP BY person_id
        ORDER BY face_count DESC
    """)
    conn.commit()

    # 5. 显示结果
    print(f"\n📊 最终统计:")
    stats = conn.execute("""
        SELECT p.id, p.face_count, p.photo_count,
               (SELECT COUNT(*) FROM faces f WHERE f.person_id = p.id) as f_cnt
        FROM persons p
        ORDER BY p.face_count DESC
    """).fetchall()
    for row in stats[:15]:
        print(f"  {row[0]}: {row[1]} faces, {row[2]} photos")
    
    total_persons = conn.execute('SELECT COUNT(*) FROM persons').fetchone()[0]
    unknown_faces = conn.execute(
        'SELECT COUNT(*) FROM faces WHERE is_unknown = 1'
    ).fetchone()[0]
    print(f"  人物总数: {total_persons}")
    print(f"  Unknown人脸: {unknown_faces}")
    print(f"  总人脸: {conn.execute('SELECT COUNT(*) FROM faces').fetchone()[0]}")

    conn.close()
    print("\n🎉 全部完成！")


if __name__ == '__main__':
    main()
