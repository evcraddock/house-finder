# Architecture

## Overview

house-finder is a single Go binary that serves two purposes:

1. **CLI** — add properties by address, rate them, leave comments, list/filter
2. **Web UI** — small read/write interface to view properties, comments, and ratings

An AI agent can drive the CLI with `--format json`. A human can use the web UI or CLI directly.

## Key Constraints

- **One API call per property.** The RapidAPI free tier is limited. The `add` command is the only thing that hits the API. Everything else reads from SQLite.
- **Single binary.** CLI commands and web server live in the same binary. `house-finder serve` starts the web UI.
- **SQLite via mattn/go-sqlite3.** Standard CGO-based SQLite binding.

## Data Model

### SQLite Schema

```sql
CREATE TABLE properties (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    address       TEXT    NOT NULL,
    mpr_id        TEXT    NOT NULL UNIQUE,
    realtor_url   TEXT    NOT NULL,
    price         INTEGER,            -- cents
    bedrooms      REAL,
    bathrooms     REAL,
    sqft          INTEGER,
    lot_size      REAL,               -- acres
    year_built    INTEGER,
    property_type TEXT,
    status        TEXT,               -- active, pending, sold
    rating        INTEGER CHECK (rating IS NULL OR (rating >= 1 AND rating <= 4)),
    raw_json      TEXT    NOT NULL,   -- full RapidAPI response
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE comments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    property_id INTEGER NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    text        TEXT    NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Why `raw_json`

The RapidAPI response contains dozens of fields (photos, tax history, schools, HOA fees, etc.). Rather than normalizing everything upfront, we store the full response and extract fields as needed. This means:

- No data loss — if we later want to display school ratings on the web UI, it's already there
- No additional API calls — parse from local data
- Simple schema — only promote fields we actively query/filter on to columns

## Data Flow

### Adding a Property

```
User: house-finder add "10109 Kay Rdg, Yukon, OK 73099"

1. Geocoder lookup (free, realtor.com suggest API)
   → returns mpr_id

2. Property ID → realtor.com URL (free, realtor.com hulk API)
   → returns href

3. RapidAPI property detail fetch (1 API call)
   → returns full property JSON

4. Parse key fields from JSON (price, beds, baths, sqft, etc.)

5. Insert into SQLite (all fields + raw_json)
   → all-or-nothing: if any step fails, nothing is saved

6. Display property summary to user
```

This is the ONLY command that hits external APIs. Steps 1-2 use free realtor.com endpoints. Step 3 uses the RapidAPI free tier.

### Everything Else

All other commands read from / write to SQLite only:

- `list` → SELECT from properties
- `show` → SELECT property + comments
- `rate` → UPDATE properties SET rating
- `comment` → INSERT into comments
- `remove` → DELETE property (cascades to comments)
- `serve` → HTTP server reading from SQLite

## CLI Design

```
house-finder add <address>           # fetch from API, store in SQLite
house-finder list [--rating N]       # list all properties, optional min rating filter
house-finder show <id>               # full property detail + comments
house-finder rate <id> <1-4>         # set rating (4 = best)
house-finder comment <id> "text"     # add a comment
house-finder comments <id>           # list comments for a property
house-finder remove <id>             # delete property and its comments
house-finder serve [--port 8080]     # start web UI
```

### Global Flags

- `--format text|json` — default `text` for humans, `json` for AI agents
- `--db <path>` — SQLite database path (default: `~/.house-finder/houses.db`)

### JSON Output

Every command outputs structured JSON when `--format json` is used. This lets an AI agent parse results reliably:

```bash
house-finder add "123 Main St" --format json | jq '.id'
house-finder list --format json | jq '.[].address'
```

## Web UI

`house-finder serve` starts an HTTP server. Three views:

1. **Property list** (`/`) — table: address, price, beds/baths/sqft, rating, link to detail
2. **Property detail** (`/property/{id}`) — key facts, rating, comments list, comment form
3. **Static assets** — embedded via `embed.FS`, no external dependencies

Tech stack:
- `net/http` standard library router
- `html/template` for rendering
- No JavaScript framework — plain HTML forms
- Templates and static files embedded in the binary

The web UI can also support adding comments and setting ratings via forms (POST endpoints).

## Project Layout

```
cmd/
  house-finder/
    main.go                 # cobra root, wires everything together

internal/
  cli/                      # cobra command definitions
    root.go                 # root command, global flags
    add.go                  # add command
    list.go                 # list command
    show.go                 # show command
    rate.go                 # rate command
    comment.go              # comment + comments commands
    remove.go               # remove command
    serve.go                # serve command

  db/                       # database layer
    db.go                   # open/init SQLite, run migrations
    migrations.go           # schema creation

  property/                 # property domain
    model.go                # Property struct
    repository.go           # CRUD operations
    service.go              # business logic (add with API, etc.)

  comment/                  # comment domain
    model.go                # Comment struct
    repository.go           # CRUD operations

  mls/                      # external API client
    client.go               # geocoder + RapidAPI calls (port of mls.sh)

  web/                      # web UI
    server.go               # HTTP server setup
    handlers.go             # route handlers
    templates/              # html/template files
      layout.html
      list.html
      detail.html
    static/                 # CSS
      style.css
```

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/mattn/go-sqlite3` | SQLite driver |
| `net/http` (stdlib) | Web server |
| `html/template` (stdlib) | HTML rendering |
| `encoding/json` (stdlib) | JSON parsing/output |

## Architectural Decisions

### ADR-1: Single binary for CLI + web

**Decision:** One binary handles both CLI commands and the web server.

**Rationale:** Simplifies deployment and distribution. `house-finder serve` just starts listening. No separate processes to manage.

### ADR-2: Store full API response as raw_json

**Decision:** Store the complete RapidAPI response in a `raw_json` TEXT column.

**Rationale:** API calls are precious (free tier). By storing everything, we can extract new fields later without re-fetching. Promoted columns (price, beds, etc.) exist only for querying and display convenience.

### ADR-3: No refresh/sync command

**Decision:** No command to re-fetch property data from the API.

**Rationale:** Protects API quota. If data needs updating, explicitly `remove` and `add` again. The friction is intentional.

### ADR-4: mattn/go-sqlite3

**Decision:** Use `github.com/mattn/go-sqlite3` for SQLite.

**Rationale:** Most widely used Go SQLite driver. Battle-tested. Standard choice.

### ADR-5: cobra for CLI

**Decision:** Use cobra for command parsing.

**Rationale:** Standard Go CLI library. Good subcommand support. Built-in help generation. Well-known to AI agents.

### ADR-6: Embedded web assets

**Decision:** Use Go's `embed.FS` to bundle templates and CSS into the binary.

**Rationale:** Single binary deployment. No file path issues. Works the same everywhere.
