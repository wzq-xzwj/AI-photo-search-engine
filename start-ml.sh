#!/bin/bash
cd /Users/wuzhaoqing/Pictures/photo-search-engine/ml-service
source venv/bin/activate
PYTHONPATH=/Users/wuzhaoqing/Pictures/photo-search-engine/ml-service:$PYTHONPATH python app/main.py
