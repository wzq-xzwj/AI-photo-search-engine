#!/usr/bin/env python3
"""Face detection for remaining photos - with proper timeouts."""
import requests
import json
import sqlite3
import sys
import os

GO_URL = "http://127.0.0.1:8080"
ML_URL = "http://127.0.0.1:8000"
DB_PATH = "/Users/wuzhaoqing/Pictures/photo-search-engine/data/photos.db"

session = requests.Session()
adapter = requests.adapters.HTTPAdapter(max_retries=0)
session.mount("http://", adapter)

def get_remaining_photos():
    conn = sqlite3.connect(DB_PATH)
    done_ids = set(r[0] for r in conn.execute("SELECT DISTINCT photo_id FROM faces").fetchall())
    conn.close()
    
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
    
    valid_exts = {'jpg', 'jpeg', 'png', 'tif', 'tiff', 'bmp', 'webp'}
    remaining = [p for p in all_photos 
                 if p["name"].lower().split('.')[-1] in valid_exts 
                 and not p["name"].startswith("._")
                 and p["id"] not in done_ids]
    return remaining

def detect_faces(photo_path):
    try:
        resp = session.post(
            f"{ML_URL}/api/v1/faces/detect",
            json={"image_path": photo_path},
            timeout=(5, 30)  # connect, read
        )
        if resp.status_code == 200:
            return resp.json()
        return None
    except:
        return None

def main():
    remaining = get_remaining_photos()
    print(f"Remaining: {len(remaining)}")
    
    conn = sqlite3.connect(DB_PATH)
    total_faces = 0
    photos_with_faces = 0
    errors = 0
    
    for i, photo in enumerate(remaining):
        photo_id = photo["id"]
        photo_path = photo["path"]
        
        result = detect_faces(photo_path)
        if result is None:
            errors += 1
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
                except: pass
            conn.commit()
            total_faces += face_count
            photos_with_faces += 1
            print(f"  [{i+1}] {photo['name']}: {face_count} faces")
        
        if (i+1) % 50 == 0:
            print(f"=== {i+1}/{len(remaining)} +faces={total_faces} errors={errors} ===")
            sys.stdout.flush()
    
    conn.close()
    print(f"\nDone! +faces={total_faces}, +photos={photos_with_faces}, errors={errors}")

if __name__ == "__main__":
    main()
