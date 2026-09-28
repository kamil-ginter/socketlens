# SocketLens Architecture

SocketLens is a local-first TCP service inspector.

```text
Browser
   |
   | HTTP on 127.0.0.1
   v
Go API + embedded UI
   |
   +--> target validation
   |      private / loopback only
   |
   +--> bounded concurrent TCP scanner
   |
   +--> snapshot diff engine
   |
   +--> SQLite history
```

## Scanner

The scanner uses a bounded worker pool and ordinary TCP connections. It does not send raw packets and does not require administrator privileges.

The first release limits a request to 1024 unique ports and accepts only loopback or private IP space.

## Storage

SQLite stores scan summaries and open ports. Before a new scan is stored, the API loads the previous snapshot for the same normalized target and computes newly-open and closed ports.

## Frontend

The frontend is plain HTML, CSS and JavaScript embedded in the Go executable. A production build therefore ships as one executable plus its local data directory.

## API

- `GET /api/health`
- `POST /api/scan`
- `GET /api/history`
