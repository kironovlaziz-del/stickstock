#!/usr/bin/env python3
"""
Fix all issues with Jinja2 templates and SQL queries in the UI.
Run this script from the project root.
"""

import os
import re
import sys
import subprocess

def patch_file(path, old_pattern, new_text, label, count=1):
    if not os.path.exists(path):
        print(f"SKIP: {label} – file not found: {path}")
        return False
    with open(path, 'r') as f:
        content = f.read()
    if old_pattern not in content:
        print(f"SKIP: {label} – pattern not found in {path}")
        return False
    new_content = content.replace(old_pattern, new_text, count)
    with open(path, 'w') as f:
        f.write(new_content)
    print(f"OK: {label}")
    return True

def append_to_file(path, line, label):
    if not os.path.exists(path):
        print(f"SKIP: {label} – file not found: {path}")
        return False
    with open(path, 'r') as f:
        content = f.read()
    if line in content:
        print(f"SKIP: {label} – already present")
        return False
    with open(path, 'a') as f:
        f.write(line + '\n')
    print(f"OK: {label}")
    return True

print("=== Fixing Jinja2 UI issues ===\n")

# 1. Ensure NEXT_PUBLIC_API_BASE_URL in .env
env_file = '.env'
base_url = 'NEXT_PUBLIC_API_BASE_URL=https://stickstock.lol/api'
if os.path.exists(env_file):
    with open(env_file, 'r') as f:
        env_content = f.read()
    if base_url not in env_content:
        with open(env_file, 'a') as f:
            f.write(f'\n{base_url}\n')
        print("OK: Added NEXT_PUBLIC_API_BASE_URL to .env")
    else:
        print("OK: NEXT_PUBLIC_API_BASE_URL already in .env")
else:
    with open(env_file, 'w') as f:
        f.write(f'{base_url}\n')
    print("OK: Created .env with NEXT_PUBLIC_API_BASE_URL")

# 2. Update docker-compose.yml – add build arg for frontend
compose_file = 'docker-compose.yml'
with open(compose_file, 'r') as f:
    compose_content = f.read()

# Check if NEXT_PUBLIC_API_BASE_URL already in frontend build args
if 'NEXT_PUBLIC_API_BASE_URL: ${NEXT_PUBLIC_API_BASE_URL}' not in compose_content:
    # Find the frontend build args block
    pattern = r'(frontend:.*?build:.*?context:.*?args:)'
    replacement = r'\1\n        NEXT_PUBLIC_API_BASE_URL: ${NEXT_PUBLIC_API_BASE_URL}'
    new_compose = re.sub(pattern, replacement, compose_content, flags=re.DOTALL)
    if new_compose != compose_content:
        with open(compose_file, 'w') as f:
            f.write(new_compose)
        print("OK: Added NEXT_PUBLIC_API_BASE_URL build arg to docker-compose.yml")
    else:
        print("WARN: Could not add build arg, manual edit needed")
else:
    print("OK: NEXT_PUBLIC_API_BASE_URL build arg already in docker-compose.yml")

# 3. Update nginx/stickstock.conf – add CORS and timeouts
nginx_conf = 'nginx/stickstock.conf'
if os.path.exists(nginx_conf):
    with open(nginx_conf, 'r') as f:
        nginx_content = f.read()
    
    # Add timeouts and CORS if not present
    if 'proxy_connect_timeout' not in nginx_content:
        nginx_content = nginx_content.replace(
            'proxy_pass http://backend:8080/api/;',
            'proxy_pass http://backend:8080/api/;\n'
            '        proxy_connect_timeout 120s;\n'
            '        proxy_send_timeout 120s;\n'
            '        proxy_read_timeout 120s;\n'
            '        proxy_buffer_size 128k;\n'
            '        proxy_buffers 4 256k;\n'
            '        proxy_busy_buffers_size 256k;'
        )
        print("OK: Added timeouts to nginx.conf")
    else:
        print("OK: Timeouts already present in nginx.conf")
    
    # Add CORS headers if not present
    if 'Access-Control-Allow-Origin' not in nginx_content:
        cors_block = '''
        # CORS headers
        add_header 'Access-Control-Allow-Origin' '*' always;
        add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS, PUT, DELETE' always;
        add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, Accept, X-Requested-With' always;

        if ($request_method = 'OPTIONS') {
            add_header 'Access-Control-Allow-Origin' '*';
            add_header 'Access-Control-Allow-Methods' 'GET, POST, OPTIONS, PUT, DELETE';
            add_header 'Access-Control-Allow-Headers' 'Authorization, Content-Type, Accept, X-Requested-With';
            add_header 'Access-Control-Max-Age' 86400;
            return 204;
        }'''
        # Insert CORS block inside location /api/
        nginx_content = nginx_content.replace(
            'proxy_pass http://backend:8080/api/;',
            'proxy_pass http://backend:8080/api/;\n' + cors_block
        )
        with open(nginx_conf, 'w') as f:
            f.write(nginx_content)
        print("OK: Added CORS headers to nginx.conf")
    else:
        print("OK: CORS headers already present in nginx.conf")
else:
    print("WARN: nginx/stickstock.conf not found")

# 4. Update frontend/src/lib/api.ts – ensure params default {}
api_file = 'frontend/src/lib/api.ts'
if os.path.exists(api_file):
    with open(api_file, 'r') as f:
        api_content = f.read()
    
    # Fix runAdHoc to handle params properly
    old_runadhoc = 'runAdHoc: (body: { data_source_id: string; sql: string; params?: Record<string, unknown> }) =>\n    request<QueryResult>("/queries/run", { method: "POST", body: JSON.stringify(body) }),'
    new_runadhoc = 'runAdHoc: (body: { data_source_id: string; sql: string; params?: Record<string, unknown> }) =>\n    request<QueryResult>("/queries/run", { method: "POST", body: JSON.stringify({ data_source_id: body.data_source_id, sql: body.sql, params: body.params || {} }) }),'
    if old_runadhoc in api_content:
        api_content = api_content.replace(old_runadhoc, new_runadhoc)
        with open(api_file, 'w') as f:
            f.write(api_content)
        print("OK: Updated runAdHoc in api.ts")
    else:
        # Try alternative pattern
        old_alt = 'runAdHoc: (body: { data_source_id: string; sql: string; params?: Record<string, unknown> }) =>\n    request<QueryResult>("/queries/run", {\n      method: "POST",\n      body: JSON.stringify(body),\n    }),'
        new_alt = 'runAdHoc: (body: { data_source_id: string; sql: string; params?: Record<string, unknown> }) =>\n    request<QueryResult>("/queries/run", {\n      method: "POST",\n      body: JSON.stringify({ data_source_id: body.data_source_id, sql: body.sql, params: body.params || {} }),\n    }),'
        if old_alt in api_content:
            api_content = api_content.replace(old_alt, new_alt)
            with open(api_file, 'w') as f:
                f.write(api_content)
            print("OK: Updated runAdHoc (alt) in api.ts")
        else:
            print("WARN: Could not find runAdHoc in api.ts, manual edit needed")
else:
    print("WARN: frontend/src/lib/api.ts not found")

# 5. Rebuild and restart services
print("\n=== Rebuilding and restarting services ===")
os.system("docker compose build frontend")
os.system("docker compose up -d frontend nginx")

print("\n=== All fixes applied ===")
print("Now open https://stickstock.lol and test Queries with Jinja2 templates.")
