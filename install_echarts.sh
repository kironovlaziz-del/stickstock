#!/bin/bash
set -e

# 1. Добавить зависимости в package.json
echo "Adding echarts dependencies to package.json..."
python3 - << 'PYEOF'
import json
import os

package_path = 'frontend/package.json'
with open(package_path, 'r') as f:
    pkg = json.load(f)

deps = pkg.get('dependencies', {})
deps['echarts'] = '^5.5.0'
deps['echarts-for-react'] = '^3.0.2'
pkg['dependencies'] = deps

with open(package_path, 'w') as f:
    json.dump(pkg, f, indent=2)

print("✅ Updated package.json")
PYEOF

# 2. Пересобрать фронтенд (автоматически выполнит npm install)
echo "Building frontend..."
docker compose build frontend
docker compose up -d frontend

echo "✅ Done! ECharts and echarts-for-react installed."
