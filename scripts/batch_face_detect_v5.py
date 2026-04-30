#!/usr/bin/env python3
"""Batch face detection v5 - with progress printing."""
import requests
import json
import sqlite3
import sys

GO_URL = "http://127.0.0.1:8080"
ML_URL = "http://127.0.0.1:8000"
DB_PATH = "/Users/wuzhaoqing/Pictures/photo-search-engine/data/photos.db"

def get_all_photos():
    all_photos = []
    page = 1
    while True:
        resp = requests.get(f"{GO_URL}/api/v1/photos?page={page}&page_size=200", timeout=10)
        if resp.status_code != 200: break
        data = resp.json()
        photos = data.get("photos", [])
        if not photos: break
        all_photos.extend(photos)
        if len(all_photos) >= data.get("total", 0): break
        page += 1
    return all_photos

def detect_faces(photo_path):
    try:
        resp = requests.post(
            f"{ML_URL}/api/v1/faces/detect",
            json={"image_path": photo_path},
            timeout=120
        )
        if resp.status_code == 200:
            return resp.json()
        return None
    except:
        return None

def main():
    photos = get_all_photos()
    print(f"Total: {len(photos)}")
    
    valid_exts = {'jpg', 'jpeg', 'png', 'tif', 'tiff', 'bmp', 'webp'}
    image_photos = [p for p in photos 
                    if p["name"].lower().split('.')[-1] in valid_exts 
                    and not p["name"].startswith("._")]
    print(f"Images: {len(image_photos)}")
    
    conn = sqlite3.connect(DB_PATH)
    
    total_faces = 0
    photos_with_faces = 0
    errors = 0
    skipped = 0
    
    for i, photo in enumerate(image_photos):
        photo_id = photo["id"]
        photo_path = photo["path"]
        photo_name = photo["name"]
        
        cursor = conn.execute("SELECT COUNT(*) FROM faces WHERE photo_id = ?", (photo_id,))
        if cursor.fetchone()[0] > 0:
            skipped += 1
            continue
        
        result = detect_faces(photo_path)
        if result is None:
            errors += 1
            if (i+1) % 20 == 0:
                print(f"[{i+1}/{len(image_photos)}] errors={errors}")
            continue
        
        face_count = result.get("face_count", 0)
        if face_count > 0:
            for face in result.get("faces", []):
                try:
                    location = json.dumps(face.get("location", {}))
                    encoding = json.dumps(face.get("encoding", []))
                    conn.execute(
                        """INSERT OR REPLACE INTO faces 
                           (photo_id, face_index, location, encoding, confidence)
                           VALUES (?, ?, ?, ?, ?)""",
                        (photo_id, face.get("index", 0), location, encoding, face.get("confidence", 0.99))
                    )
                except Exception as e:
                    pass
            conn.commit()
            total_faces += face_count
            photos_with_faces += 1
            print(f"  [{i+1}] {photo_name}: {face_count} faces")
        
        if (i+1) % 50 == 0:
            print(f"=== {i+1}/{len(image_photos)} faces={total_faces} photos={photos_with_faces} errors={errors} skipped={skipped} ===")
    
    conn.close()
    print(f"\nDone! faces={total_faces}, photos={photos_with_faces}, errors={errors}, skipped={skipped}")

if __name__ == "__main__":
    main()
