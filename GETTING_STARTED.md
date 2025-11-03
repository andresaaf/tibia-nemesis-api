# Getting Started with Tibia Nemesis API

This guide shows how to run the Go API server.

## Quick Start

```powershell
cd tibia-nemesis-api

# Install dependencies
go mod tidy

# Run the server (uses port 8080 by default)
go run .\cmd\server\main.go
```

The API will:
- Start HTTP server on port 8080
- Create SQLite database `tibia-nemesis-api.db`
- Load boss metadata from `bosses_metadata.yaml`
- Schedule daily refresh at 09:00 CET
- Log startup information

## Test the API

```powershell
# Health check
curl "http://localhost:8080/api/v1/status"

# Trigger a manual refresh for Antica world
curl -X POST "http://localhost:8080/api/v1/refresh?world=Antica"

# Get spawn data for Antica (pre-filtered by inclusion_range)
curl "http://localhost:8080/api/v1/spawnables?world=Antica"

# List all worlds with data
curl "http://localhost:8080/api/v1/worlds"

# Get boss history
curl "http://localhost:8080/api/v1/bosses/Furyosa/history?world=Antica"
```

## Configuration

### Environment Variables

- `PORT` - HTTP server port (default: 8080)
- `DB_PATH` - SQLite database path (default: tibia-nemesis-api.db)
- `REFRESH_AT` - Daily refresh time HH:MM (default: 09:00)
- `TZ` - Timezone for scheduler (default: CET)

Example:
```powershell
$env:PORT="9000"
$env:DB_PATH="./data/tibia.db"
$env:REFRESH_AT="08:00"
$env:TZ="America/New_York"
```

## Boss Metadata

The API uses `bosses_metadata.yaml` to apply inclusion_range filtering:

- **min_days**: Boss is hidden if days since last kill < min_days
- **max_days**: Boss is always shown if days since last kill >= max_days  
- **No inclusion_range**: Boss is always shown regardless of days

Example:
```yaml
bosses:
  Furyosa:
    name: Furyosa
    inclusion_range:
      min_days: 12
      max_days: 46
```

This ensures the API only returns bosses that are likely to spawn based on their kill history.

## Troubleshooting

### API not scraping data

1. Check internet connection
2. Verify tibia-statistic.com is accessible
3. Check API logs for HTTP errors
4. Try manual refresh: `curl -X POST "http://localhost:8080/api/v1/refresh?world=Antica"`

### Database errors

1. Delete `tibia-nemesis-api.db` and restart API
2. Check file permissions
3. Ensure directory is writable

### Metadata not loading

1. Verify `bosses_metadata.yaml` exists in the same directory as the binary
2. Check YAML syntax is valid
3. Review API startup logs for metadata loading errors

## Development

### Modify Scraping Logic

1. Edit `internal/scraper/scraper.go`
2. Update regex patterns if tibia-statistic.com HTML changes
3. Restart API to apply changes
4. Test with manual refresh endpoint

### Add New Endpoints

1. Add handler in `internal/http/handlers.go`
2. Register route in `internal/http/router.go`
3. Update README with new endpoint docs

### Update Boss Metadata

1. Edit `bosses_metadata.yaml` manually
2. Add/update inclusion_range values based on spawn patterns
3. Restart API to reload metadata

## Deployment

For production deployment:

1. Build binary: `go build -o tibia-nemesis-api.exe .\cmd\server\main.go`
2. Ensure `bosses_metadata.yaml` is in same directory as binary
3. Set production environment variables
4. Run as a service/daemon
5. Configure reverse proxy (nginx/caddy) if needed
6. Set up monitoring and logging

## API Documentation

See [README.md](README.md) for complete API endpoint documentation.
