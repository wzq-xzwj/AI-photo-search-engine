#!/usr/bin/env python3
"""Batch face detection for all photos in the database."""
import sqlite3
import requests
import json
import time
import sys

DB_PATH = "/Users/wuzhaoqing/Pictures/photo-search-engine/data/photos.db"
ML_URL = "http://127.0.0.1:8000"

def get_all_photos():
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.execute("SELECT id, path FROM photos ORDER BY id")
    photos = cursor.fetchall()
    conn.close()
    return photos

def detect_faces(photo_id, photo_path):
    """Call ML service to detect faces."""
    try:
        resp = requests.post(
            f"{ML_URL}/api/v1/faces/detect",
            json={"image_path": photo_path},
            timeout=60
        )
        if resp.status_code == 200:
            return resp.json()
        else:
            return None
    except Exception as e:
        return None

def save_face_result(conn, photo_id, photo_path, face_data):
    """Save face detection results to database."""
    if not face_data or face_data.get("face_count", 0) == 0:
        return 0
    
    count = 0
    for face in face_data.get("faces", []):
        try:
            conn.execute(
                """INSERT OR REPLACE INTO faces 
                   (photo_id, photo_path, face_index, location_top, location_right, 
                    location_bottom, location_left, encoding, confidence)
                   VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)""",
                (
                    photo_id,
                    photo_path,
                    face.get("index", 0),
                    face["location"]["top"],
                    face["location"]["right"],
                    face["location"]["bottom"],
                    face["location"]["left"],
                    json.dumps(face.get("encoding", [])),
                    face.get("confidence", 0)
                )
            )
            count += 1
        except Exception as e:
            print(f"  Error saving face: {e}")
    conn.commit()
    return count

def ensure_faces_table(conn):
    """Create faces table if it doesn't exist."""
    conn.execute("""
        CREATE TABLE IF NOT EXISTS faces (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            photo_id TEXT NOT NULL,
            photo_path TEXT NOT NULL,
            face_index INTEGER DEFAULT 0,
            location_top INTEGER,
            location_right INTEGER,
            location_bottom INTEGER,
            location_left INTEGER,
            encoding TEXT,
            confidence REAL,
            person_id TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
            UNIQUE(photo_id, face_index)
        )
    """)
    conn.commit()

def main():
    photos = get_all_photos()
    print(f"Total photos: {len(photos)}")
    
    conn = sqlite3.connect(DB_PATH)
    ensure_faces_table(conn)
    
    total_faces = 0
    photos_with_faces = 0
    errors = 0
    
    for i, (photo_id, photo_path) in enumerate(photos):
        # Skip non-image files
        ext = photo_path.lower().split('.')[-1]
        if ext not in ('jpg', 'jpeg', 'png', 'tif', 'tiff', 'bmp', 'webp'):
            continue
        
        # Check if already detected
        cursor = conn.execute("SELECT COUNT(*) FROM faces WHERE photo_id = ?", (photo_id,))
        if cursor.fetchone()[0] > 0:
            continue
        
        result = detect_faces(photo_id, photo_path)
        if result is None:
            errors += 1
            if (i+1) % 50 == 0:
                print(f"Progress: {i+1}/{len(photos)}, faces={total_faces}, errors={errors}")
            continue
        
        face_count = result.get("face_count", 0)
        if face_count > 0:
            saved = save_face_result(conn, photo_id, photo_path, result)
            total_faces += saved
            photos_with_faces += 1
            print(f"  [{i+1}] {photo_path.split('/')[-1]}: {face_count} faces")
        
        if (i+1) % 50 == 0:
            print(f"Progress: {i+1}/{len(photos)}, faces={total_faces}, photos_with_faces={photos_with_faces}, errors={errors}")
    
    conn.close()
    print(f"\nDone! Total faces: {total_faces}, Photos with faces: {photos_with_faces}, Errors: {errors}")

if __name__ == "__main__":
    main()
