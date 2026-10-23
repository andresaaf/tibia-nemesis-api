# tibia-nemesis-api

We Vibin'
A small Go REST API that scrapes Tibia boss data, applies spawn chance logic, and stores results for consumption by the Discord bot.

## Features
- Daily refresh (09:30 by default) and on-demand refresh endpoint
- SQLite persistence of computed spawn percentages per world
- Simple HTTP/JSON endpoints for the bot

## Endpoints
- `GET /api/v1/status` - Health check
- `GET /api/v1/worlds` - List all worlds with data
- `GET /api/v1/bosses?world=Antica` - Get all bosses with spawnable status
- `GET /api/v1/boss/{name}/history?world=Antica` - Get boss history
- `POST /api/v1/refresh?world=Antica` - Trigger manual data refresh

### Response Format

**`/api/v1/bosses`** returns:
```json
{
  "world": "Antica",
  "updated_at": "2025-11-06T18:30:00Z",
  "bosses": [
    {
      "name": "Rukor Zad",
      "percent": null,
      "days_since_kill": 36,
      "spawnable": true
    },
    {
      "name": "Hirintror",
      "percent": null,
      "days_since_kill": 14,
      "spawnable": true
    }
  ]
}
```

## Quick start

```powershell
# From tibia-nemesis-api folder
$env:PORT="8080"
$env:DB_PATH="tibia-nemesis-api.db"
$env:REFRESH_AT="09:30"
$env:TZ="Europe/Berlin"
$env:SCRAPER="guildstats"   # or "tibia-statistic"

go run ./cmd/server
```

## Notes
- Boss data is scraped from [guildstats.eu](https://guildstats.eu/bosses) by default. The tibia-statistic.com scraper is kept as `SCRAPER=tibia-statistic`, but that site currently blocks automated requests with a Cloudflare challenge.
- guildstats.eu boss names are mapped to `bosses_metadata.yaml` names in `internal/scraper/guildstats.go` (e.g. `draptors` → `Draptor`, and `Fleabringer` data is copied to its location variants).
- Bosses the source no longer reports are removed from the database on each refresh.
- Percentages are capped to integers and may be null (unknown) when not determinable.
- Boss filtering uses `bosses_metadata.yaml` with inclusion_range rules:
  - **min_days**: Boss is hidden if days since last kill < min_days
  - **max_days**: Boss is always shown if days since last kill >= max_days
  - No range defined: Boss is always shown regardless of days
- To update boss metadata, modify `Bosses.py` in the Discord bot repo, then run `py export_bosses_metadata.py` to regenerate the YAML file.
