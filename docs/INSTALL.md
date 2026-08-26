# Installation Guide

## Prerequisites

- VPS with Linux (Ubuntu 22.04 or later)
- Docker and Docker Compose installed
- Domain name (optional, but recommended for HTTPS)
- Supabase account (free tier works) – for Auth and Postgres

---

## Step 1: Clone the repository

```bash
git clone https://github.com/your-username/stickstock.git
cd stickstock

cp .env.example .env
nano .env

DATABASE_URL – your Supabase Postgres connection string.

JWT_SECRET – a strong secret (generate with openssl rand -base64 32).

SUPABASE_URL and SUPABASE_ANON_KEY – from your Supabase project.

ENCRYPTION_KEY – generate with openssl rand -base64 32

Step 3: Run database migrations

Apply the SQL files in backend/migrations/ using Supabase SQL Editor or psql.
Step 4: Build and start the containers

docker compose build
docker compose up -d

Step 5: Configure nginx with SSL (optional)

If you have a domain, obtain Let's Encrypt certificates:
docker compose stop nginx
certbot certonly --standalone -d yourdomain.com -d www.yourdomain.com
docker compose up -d nginx

Update nginx/stickstock.conf with your domain.
Step 6: Access the platform

Open your browser at https://yourdomain.com (or http://server-ip).

Troubleshooting

    Check logs: docker compose logs backend, docker compose logs frontend

    Database connection errors: verify DATABASE_URL in .env

For more help, refer to User Guide.
