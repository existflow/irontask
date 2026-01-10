# Zero-Cost Sync Hosting Guide

You can host your own IronTask Sync Server for **free** using [Supabase](https://supabase.com) (Database) and [Render](https://render.com) (Server).

## Prerequisites
- GitHub account
- Supabase account (Free tier)
- Render account (Free tier)

## Step 1: Create Supabase Database
1. Create a new project on Supabase.
2. Go to **Project Settings** -> **Database**.
3. Copy the **Connection String (URI)**. It looks like:
   `postgres://postgres:[YOUR-PASSWORD]@db.xxx.supabase.co:5432/postgres`

## Step 2: Deploy IronTask Server to Render
1. Go to [Render Dashboard](https://dashboard.render.com).
2. Click **New +** -> **Web Service**.
3. Connect your fork of the `irontask` repository (or use the public Docker image if available).
4. Settings:
   - **Environment**: Docker
   - **Region**: Closest to you
   - **Instance Type**: Free
5. **Environment Variables**:
   - `DATABASE_URL`: Paste your Supabase connection string.
   - `PORT`: 8080 (Render detects this, but good to be explicit).
6. Click **Create Web Service**.
7. Wait for deployment. You will get a URL like `https://irontask-sync.onrender.com`.

## Step 3: Configure Client
On your computer:

```bash
# Register with your new server
task sync register --server https://irontask-sync.onrender.com

# Or manually config
task sync config --server https://irontask-sync.onrender.com
```

## Alternative: Self-Host with Docker
If you have a VPS:

```yaml
version: '3'
services:
  app:
    image: tphuc/irontask-server:latest
    environment:
      - DATABASE_URL=postgres://user:pass@supabase-host:5432/db
    ports:
      - "8080:8080"
```
