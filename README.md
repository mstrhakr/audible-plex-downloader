# Audible Plex Downloader

A pure Go Docker application that authenticates with Audible, downloads audiobooks, removes DRM via FFmpeg, fetches enriched metadata from Audnexus, and organizes files in Plex-compatible `Author/Title/Title.m4b` structure.

## Quick Start

```bash
# Clone the repo
git clone https://github.com/nick/audible-plex-downloader.git
cd audible-plex-downloader

# Copy and edit config
cp config.example.yaml config/config.yaml

# Start with Docker Compose
docker compose up -d
```

Then visit `http://localhost:8080` to authenticate and manage your library.

## Configuration

Configuration can be provided via `config.yaml` or environment variables:

| Env Variable | Default | Description |
|---|---|---|
| `DATABASE_TYPE` | `sqlite` | Database backend (`sqlite` or `postgres`) |
| `DATABASE_PATH` | `/config/audible.db` | SQLite database path |
| `DATABASE_DSN` | | PostgreSQL connection string |
| `AUDIOBOOKS_PATH` | `/audiobooks` | Output directory (Plex library root) |
| `DOWNLOADS_PATH` | `/downloads` | Temporary download directory |
| `CONFIG_PATH` | `/config` | Config/auth storage directory |
| `OUTPUT_FORMAT` | `m4b` | Output format (`m4b` or `mp3`) |
| `DOWNLOAD_CONCURRENCY` | `0` | Concurrent downloads (0 = auto-detect based on CPU) |
| `PLEX_URL` | | Plex server URL for library scan triggers |
| `PLEX_TOKEN` | | Plex authentication token |
| `SYNC_SCHEDULE` | `0 */6 * * *` | Cron schedule for library sync |

## Docker Compose

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
      - DATABASE_TYPE=sqlite
    restart: unless-stopped
```

## Output Structure

Files are organized for Plex audiobook libraries:

```
/audiobooks/
  Author Name/
    Book Title/
      Book Title.m4b
      Book Title.chapters.txt
      cover.jpg
```

## Development

```bash
# Build locally
go build -o audible-plex-downloader ./cmd/server

# Run
./audible-plex-downloader
```

Requires Go 1.22+ and CGO (for SQLite).

## License

MIT
