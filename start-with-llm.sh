#!/bin/bash
# Start script with LLM API configuration

cd /Users/wuzhaoqing/Pictures/photo-search-engine

# Export LLM configuration
export LLM_PROVIDER=openai
export OPENAI_BASE_URL=https://api.deepseek.com/v1
export OPENAI_API_KEY=sk-13d33f25229248d28915daf3761ff0e7
export OPENAI_MODEL=deepseek-chat

# Start the server
./photo-server
