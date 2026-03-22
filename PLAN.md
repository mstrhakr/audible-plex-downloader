# Audible Plex Downloader

A pure Go Docker application that authenticates with Audible, downloads audiobooks, removes DRM via FFmpeg, fetches enriched metadata from Audnexus, and organizes files in Plex-compatible `Author/Title/Title.m4b` structure.

## Architecture

```text
┌─────────────────────────────────────────────────────────────────┐
│                    Docker Container (~80MB)                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  Go Binary   │  │   FFmpeg     │  │   Alpine Linux       │  │
│  │   (~15MB)    │  │   (~30MB)    │  │      (~5MB)          │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
├─────────────────────────────────────────────────────────────────┤
│                         Volumes                                  │
│  /config (auth, db)  │  /audiobooks (Plex)  │  /downloads (temp)│
└─────────────────────────────────────────────────────────────────┘
```

### Components

| Component | Technology | Purpose |
| --------- | ---------- | ------- |
| Auth & API | Pure Go (`go-audible` lib) | OAuth, device registration, Audible API |
| Web UI | HTMX + Go templates | Dashboard, library browser, settings |
| Audio Processing | FFmpeg (subprocess) | AAX decryption, format conversion |
| Metadata | Audnexus API | Enriched book/author/chapter data |
| Database | SQLite (default) / PostgreSQL | Library state, queue, settings |
| Scheduler | `robfig/cron` | Automated library sync |

---

## Features

- **Authentication**: Browser-based OAuth flow with Audible/Amazon
- **Library Sync**: Automatic detection of new purchases (scheduled or manual)
- **Download Queue**: Concurrent downloads with progress tracking
- **DRM Removal**: FFmpeg-based decryption (AAX activation bytes, AAXC voucher)
- **Output Formats**: M4B (single file) or MP3 (chapter-split), configurable
- **Metadata**: Audnexus-enriched tags embedded in audio files
- **Plex Structure**: `{Author}/{Title}/{Title}.m4b` + `{Title}.chapters.txt`
- **Web Dashboard**: Real-time progress via SSE, library browser, settings

---

## Implementation Phases

### Phase 1: Project Foundation (2 days)

1. **Initialize Go module**

   ```bash
   go mod init github.com/nick/audplexus
   ```

2. **Create Dockerfile**
   - Multi-stage build: `golang:1.22-alpine` → `alpine:3.19`
   - Install FFmpeg, ca-certificates
   - Expose port 8080

3. **Set up docker-compose.yml**

   ```yaml
   services:
     audible-plex:
       build: .
       ports:
         - "8080:8080"
       volumes:
         - ./config:/config
         - /path/to/audiobooks:/audiobooks
         - ./downloads:/downloads
       environment:
         - DATABASE_TYPE=sqlite  # or postgres
   ```

4. **Database abstraction**
   - Interface supporting SQLite and PostgreSQL
   - Tables: `books`, `download_queue`, `sync_history`, `settings`, `devices`
   - Migrations via `golang-migrate`

### Phase 2: Audible Library Integration (depends on go-audible)

1. **Import go-audible library**
   - OAuth flow integration
   - Credential storage (encrypted)
   - API client for library/download operations

2. **Library sync service**
   - Fetch full library with pagination
   - Compare with local DB, identify new titles
   - Queue new books for download

### Phase 3: Audio Processing Pipeline (2 days)

1. **Download manager**
   - Concurrent downloads (configurable, default 2)
   - Progress tracking
   - Resumable downloads

2. **DRM removal pipeline**
   - FFmpeg wrapper for AAX: `ffmpeg -activation_bytes <bytes> -i input.aax -c copy output.m4b`
   - FFmpeg wrapper for AAXC: `ffmpeg -audible_key <key> -audible_iv <iv> -i input.aaxc -c copy output.m4b`
   - Integrity validation (duration check)

3. **Format conversion**
   - M4B: Direct copy from decrypted (fastest)
   - MP3: Chapter-split using FFmpeg + chapters metadata

4. **Cover art processing**
    - Download high-res cover
    - Embed in M4B
    - Save standalone `cover.jpg`

### Phase 4: Metadata & Audnexus (1 day)

1. **Audnexus API client**
    - `GET /books/{asin}` - Book metadata
    - `GET /authors/{asin}` - Author details
    - `GET /chapters/{asin}` - Chapter info
    - Fallback to Audible data if unavailable

2. **Metadata processor**
    - Merge Audnexus + Audible (Audnexus preferred)
    - Fields: title, author, narrator, series, position, description, genres

3. **File tagger**
    - Embed via FFmpeg metadata options
    - ID3/MP4 tags

### Phase 5: Plex Organization (1 day)

1. **File organizer**
    - Structure: `{library_root}/{Author}/{Title}/{Title}.{ext}`
    - Sanitize filenames (remove `<>:"/\|?*`)
    - Series handling: `{Title} - {Series}, Book {N}`

2. **Chapter file generation**
    - `{Title}.chapters.txt` format:

      ```text
      00:00:00.000 Opening Credits
      00:01:23.456 Chapter 1
      00:45:12.789 Chapter 2
      ...
      ```

3. **Post-processing**
    - Clean temp files
    - Update database
    - Optional: Trigger Plex library scan via API

### Phase 6: Web UI (2 days)

1. **HTMX-based interface**
    - Dashboard: overview, recent downloads, sync status
    - Library browser: searchable/sortable table
    - Book detail: metadata, cover, actions
    - Download queue: progress bars (SSE)
    - Settings: auth, output format, paths, schedule

2. **Authentication UI**
    - `/auth/start` - Redirect to Amazon OAuth
    - `/auth/callback` - Receive authorization code
    - `/auth/status` - Current auth state

### Phase 7: Scheduling (1 day)

1. **Cron scheduler**
    - Library sync: hourly/daily/weekly/disabled
    - Manual trigger via UI
    - Quiet hours configuration

2. **Notifications** (future)
    - Webhook on download complete
    - Discord/Slack integration

---

## Project Structure

```text
audplexus/
├── cmd/
│   └── server/
│       └── main.go                 # Entry point
├── internal/
│   ├── audnexus/
│   │   └── client.go               # Audnexus API client
│   ├── audio/
│   │   ├── ffmpeg.go               # FFmpeg wrapper
│   │   ├── decrypt.go              # DRM removal
│   │   ├── convert.go              # Format conversion
│   │   └── tag.go                  # Metadata embedding
│   ├── database/
│   │   ├── interface.go            # DB abstraction
│   │   ├── sqlite.go
│   │   ├── postgres.go
│   │   ├── models.go               # Data models
│   │   └── migrations/
│   │       └── 001_initial.sql
│   ├── library/
│   │   ├── sync.go                 # Library sync logic
│   │   └── download.go             # Download manager
│   ├── organizer/
│   │   └── plex.go                 # File organization
│   ├── scheduler/
│   │   └── cron.go                 # Scheduled jobs
│   └── web/
│       ├── server.go               # HTTP server
│       ├── handlers/
│       │   ├── auth.go
│       │   ├── library.go
│       │   ├── downloads.go
│       │   └── settings.go
│       ├── templates/
│       │   ├── base.html
│       │   ├── dashboard.html
│       │   ├── library.html
│       │   ├── auth.html
│       │   └── settings.html
│       └── static/
│           ├── htmx.min.js
│           └── style.css
├── Dockerfile
├── docker-compose.yml
├── go.mod
├── go.sum
├── config.example.yaml
├── PLAN.md
└── README.md
```

---

## Dependencies

| Package | Purpose |
| --- | --- |
| `github.com/mstrhakr/go-audible` | Audible auth & API |
| `github.com/gin-gonic/gin` | HTTP router |
| `github.com/mattn/go-sqlite3` | SQLite driver |
| `github.com/lib/pq` | PostgreSQL driver |
| `github.com/golang-migrate/migrate/v4` | Migrations |
| `github.com/robfig/cron/v3` | Scheduler |
| `github.com/rs/zerolog` | Logging |

---

## Configuration

```yaml
# config.yaml
server:
  port: 8080
  
database:
  type: sqlite  # sqlite | postgres
  path: /config/audible.db  # for sqlite
  # dsn: postgres://user:pass@host/db  # for postgres

paths:
  audiobooks: /audiobooks
  downloads: /downloads
  config: /config

output:
  format: m4b  # m4b | mp3
  embed_cover: true
  chapter_file: true

sync:
  schedule: "0 */6 * * *"  # every 6 hours
  enabled: true

plex:
  url: http://plex:32400
  token: ""  # optional, for library scan trigger
```

---

## External Dependencies

- **go-audible**: Pure Go Audible authentication and API library (separate repo)
- **FFmpeg**: Audio processing (bundled in Docker image)

---

## Verification Checklist

- [ ] Docker image builds successfully (< 100MB)
- [ ] OAuth flow completes in browser
- [ ] Library sync detects all owned books
- [ ] AAX download and decryption works
- [ ] AAXC download and decryption works
- [ ] Metadata embedded correctly
- [ ] Files organized as `{Author}/{Title}/{Title}.m4b`
- [ ] Chapter file generated
- [ ] Plex detects and displays audiobook
- [ ] Audnexus metadata shows in Plex (via audnexus plugin)
- [ ] Scheduled sync runs on time
- [ ] Web UI functional on mobile

---

## Future Enhancements

- Multi-region support (UK, DE, FR, etc.)
- Wishlist sync and purchase notifications
- Audiobookshelf compatibility mode
- Podgrab-style podcast support (Audible podcasts)
- Discord/Slack notifications
- Bulk re-download with updated metadata

