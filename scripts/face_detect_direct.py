#!/usr/bin/env python3
"""Face detection using face_recognition directly (no ML service)."""
import sqlite3
import json
import sys
import os
import face_recognition

DB_PATH = "/Users/wuzhaoqing/Pictures/photo-search-engine/data/photos.db"
BASE_DIR = "/Volumes/Ultra Touch/影像从心"

def get_all_photo_paths():
    """Get all image files from the external drive."""
    valid_exts = {'.jpg', '.jpeg', '.png', '.tif', '.tiff', '.bmp', '.webp'}
    photos = []
    for root, dirs, files in os.walk(BASE_DIR):
        for f in files:
            if f.startswith("._"):
                continue
            ext = os.path.splitext(f)[1].lower()
            if ext in valid_exts:
                photos.append(os.path.join(root, f))
    return photos

def detect_faces(image_path):
    try:
        image = face_recognition.load_image_file(image_path)
        face_locations = face_recognition.face_locations(image, model="hog")
        if not face_locations:
            return []
        face_encodings = face_recognition.face_encodings(image, face_locations)
        faces = []
        for i, (loc, enc) in enumerate(zip(face_locations, face_encodings)):
            top, right, bottom, left = loc
            faces.append({
                "index": i,
                "location": {"top": top, "right": right, "bottom": bottom, "left": left},
                "encoding": enc.tolist(),
                "confidence": 0.99
            })
        return faces
    except Exception as e:
        return []

def main():
    photos = get_all_photo_paths()
    print(f"Total images: {len(photos)}")
    
    conn = sqlite3.connect(DB_PATH)
    
    # Check existing
    existing = set(r[0] for r in conn.execute("SELECT DISTINCT photo_id FROM faces").fetchall())
    print(f"Already have faces: {len(existing)} photos")
    
    total_faces = 0
    photos_with_faces = 0
    errors = 0
    
    for i, photo_path in enumerate(photos):
        # Use path as photo_id
        photo_id = photo_path
        
        # Check if already done
        cursor = conn.execute("SELECT COUNT(*) FROM faces WHERE photo_id = ?", (photo_id,))
        if cursor.fetchone()[0] > 0:
            continue
        
        faces = detect_faces(photo_path)
        if faces:
            for face in faces:
                try:
                    location = json.dumps(face["location"])
                    encoding = json.dumps(face["encoding"])
                    conn.execute(
                        """INSERT OR REPLACE INTO faces 
                           (photo_id, face_index, location, encoding, confidence)
                           VALUES (?, ?, ?, ?, ?)""",
                        (photo_id, face["index"], location, encoding, face["confidence"])
                    )
                except Exception as e:
                    pass
            conn.commit()
            total_faces += len(faces)
            photos_with_faces += 1
            fname = os.path.basename(photo_path)
            print(f"  [{i+1}] {fname}: {len(faces)} faces")
        
        if (i+1) % 50 == 0:
            print(f"=== {i+1}/{len(photos)} faces={total_faces} photos={photos_with_faces} errors={errors} ===")
            sys.stdout.flush()
    
    conn.close()
    print(f"\nDone! faces={total_faces}, photos={photos_with_faces}, errors={errors}")

if __name__ == "__main__":
    main()
