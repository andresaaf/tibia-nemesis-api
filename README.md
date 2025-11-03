# tibia-nemesis-api

A small Go REST API that scrapes Tibia boss data, applies spawn chance logic, and stores results for consumption by the Discord bot.

## Features
- Daily refresh (09:00 by default) and on-demand refresh endpoint
- SQLite persistence of computed spawn percentages per world
- Simple HTTP/JSON endpoints for the bot

## Endpoints
- GET /api/v1/status
- GET /api/v1/worlds
- GET /api/v1/spawnables?world=Antica
- GET /api/v1/bosses/{name}/history?world=Antica
- POST /api/v1/refresh?world=Antica

## Quick start

```powershell
# From tibia-nemesis-api folder
$env:PORT="8080"
$env:DB_PATH="tibia-nemesis-api.db"
$env:REFRESH_AT="09:00"
$env:TZ="Europe/Berlin"

go run ./cmd/server
```

## Notes
- The default scraper uses goquery; selectors are left as TODOs and may require tuning.
- Percentages are capped to integers and may be null (unknown) when not determinable.
