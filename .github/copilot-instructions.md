# Vortludo - AI Coding Guidelines

## Architecture Overview

Vortludo is a Go web application implementing a Wordle-style word guessing game. The architecture follows a clean separation:

-   **Backend**: Gin HTTP server with in-memory session storage
-   **Frontend**: HTMX + Alpine.js for dynamic updates, Bootstrap for styling
-   **Data**: JSON word lists with hints, text file for accepted words
-   **Game Logic**: Server-side validation with client-side UI updates

## Key Components

-   `cmd/vortludo/main.go`: Server setup, routing, middleware chain
-   `internal/game/`: Core game logic (word selection, guess validation, scoring)
-   `internal/handlers/`: HTTP endpoints with HTMX-aware responses
-   `internal/session/`: In-memory session management with cleanup goroutines
-   `internal/middleware/`: CSRF protection, rate limiting, security headers
-   `internal/models/`: Data structures (App, GameState, WordEntry)
-   `static/client.js`: Alpine.js game logic and HTMX interactions
-   `templates/`: Go HTML templates with partials

## Critical Patterns

### Session Management

-   Sessions stored in-memory with `sync.RWMutex` protection
-   Automatic cleanup of expired sessions every 10 minutes
-   Session ID in secure HTTP-only cookies
-   Game state persists across requests within session timeout

### Game Logic

-   Words loaded from `data/words.json` (with hints) and `data/accepted_words.txt`
-   5-letter words, 6 guesses maximum
-   Guess validation: must be accepted word, not previously guessed
-   Scoring: correct/present/absent with Wordle-style color coding
-   Rune buffer pooling for performance optimization

### Security & Middleware

-   CSRF tokens required on all POST requests
-   Rate limiting per IP (configurable RPS/burst)
-   Content Security Policy with CDN allowances
-   Request ID tracking for logging correlation

### HTMX Integration

-   Server returns HTML fragments for dynamic updates
-   `HX-Trigger` headers for client-side events
-   Graceful degradation to full page loads when JS disabled

### Configuration

-   Environment variables with `godotenv` loading
-   Production vs development asset serving (`dist/` vs `static/`)
-   Configurable timeouts, cache ages, rate limits

## Development Workflow

### Building & Running

```bash
go build ./cmd/vortludo
./vortludo
# Server starts on http://localhost:8080
```

### Testing

```bash
go test ./...
# Tests in internal/*/test/ directories
```

### Key Conventions

-   Use `util.LogInfo/Warn/Fatal` for structured logging with request IDs
-   Context propagation for cancellation and request tracking
-   Error codes from `constants` package for client communication
-   Functional options pattern for configuration
-   Pool-based memory management for performance-critical paths

## Common Tasks

### Adding New Game Features

1. Extend `models.GameState` if needed
2. Add validation logic in `game` package
3. Create/update handlers in `handlers` package
4. Update client.js for UI interactions
5. Add HTML partials in `templates/partials/`

### Modifying Word Lists

-   Valid target words: `data/words.json` (with hints)
-   Accepted guesses: `data/accepted_words.txt`
-   Both loaded at startup and cached in memory

### Session/Data Persistence

-   Currently in-memory only
-   Add database layer in `internal/` if needed
-   Maintain existing session interface compatibility

### Frontend Changes

-   Use Alpine.js `x-data` for component state
-   HTMX attributes for server interactions
-   Bootstrap classes for responsive design
-   Test both HTMX and full-page fallback paths
