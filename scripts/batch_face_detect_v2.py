#!/usr/bin/env python3
"""Batch face detection via Go backend API."""
import requests
import json
import time
import sys

GO_URL = "http://127.0.0.1:8080"
ML_URL = "http://127.0.0.1:8000"

def get_all_photos(page=1, page_size=100):
    """Get photos from Go backend."""
    resp = requests.get(f"{GO_URL}/api/v1/photos?page={page}&page_size={page_size}", timeout=10)
    if resp.status_code == 200:
        data = resp.json()
        return data.get("photos", []), data.get("total", 0)
    return [], 0

def detect_faces_via_ml(photo_path):
    """Call ML service directly to detect faces."""
    try:
        resp = requests.post(
            f"{ML_URL}/api/v1/faces/detect",
            json={"image_path": photo_path},
            timeout=60
        )
        if resp.status_code == 200:
            return resp.json()
        return None
    except Exception as e:
        return None

def detect_faces_via_go(photo_id):
    """Call Go backend to detect faces."""
    try:
        resp = requests.post(
            f"{GO_URL}/api/v1/faces/detect",
            json={"photo_id": photo_id},
            timeout=60
        )
        if resp.status_code == 200:
            return resp.json()
        else:
            return None
    except Exception as e:
        return None

def main():
    # Get total photo count
    _, total = get_all_photos(1, 1)
    print(f"Total photos: {total}")
    
    # Process in batches
    page_size = 100
    total_faces = 0
    photos_with_faces = 0
    errors = 0
    skipped = 0
    
    for page in range(1, (total // page_size) + 2):
        photos, _ = get_all_photos(page, page_size)
        if not photos:
            break
        
        for photo in photos:
            photo_id = photo["id"]
            photo_path = photo["path"]
            photo_name = photo["name"]
            
            # Skip non-image files
            ext = photo_name.lower().split('.')[-1]
            if ext not in ('jpg', 'jpeg', 'png', 'tif', 'tiff', 'bmp', 'webp'):
                skipped += 1
                continue
            
            # Try Go backend face detection
            result = detect_faces_via_go(photo_id)
            if result is None:
                errors += 1
                continue
            
            face_count = result.get("face_count", 0)
            if face_count > 0:
                total_faces += face_count
                photos_with_faces += 1
                print(f"  {photo_name}: {face_count} faces")
        
        print(f"Page {page} done, faces={total_faces}, photos_with_faces={photos_with_faces}, errors={errors}")
    
    print(f"\nDone! Total faces: {total_faces}, Photos with faces: {photos_with_faces}, Errors: {errors}, Skipped: {skipped}")

if __name__ == "__main__":
    main()
