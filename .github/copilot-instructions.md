# AI Coding Agent Guide for `vortludo`

Concise, project-specific instructions to get productive quickly. Focus on existing patterns – not aspirations.

## Big Picture

Server-side Wordle-style game in Go using `gin`. Stateless HTTP endpoints backed by in-memory session + game state maps with periodic cleanup. Assets served from `static/` & `templates/` (or `dist/` in production). No database; persistence lasts only for session lifetime.

## Core Types & Data

-   `App` (in `types.go`): central container (word lists, hint map, sessions, rate limiters, config timings, gzip rune pool).
-   `GameState`: tracks guesses (`[][]GuessResult`), progress (`CurrentRow`, `Won`, `GameOver`), `SessionWord` vs revealed `TargetWord`.
-   Word sources: `data/words.json` (objects `{word,hint}`) filtered to length 5; `data/accepted_words.txt` (uppercase canonical forms allowed even if not target).
-   Constants in `constants.go` define game constraints, error codes, routes.

## Request Flow

1. Middleware stack (`main.go`): request ID → security headers → CSRF cookie issue/validate → gzip → cache headers (prod vs dev) → rate limiting (attached per mutating route).
2. Session acquired/created via cookie `session_id` (`session.go`). In-memory only; each request updates `LastAccessTime`.
3. Handlers:
    - `homeHandler`: render full page `index.html`.
    - `newGameHandler`: reset session or reuse, supports HTMX partial refresh. Accepts optional `completedWords` JSON array to avoid repeats.
    - `guessHandler`: validates state, normalizes guess, rejects duplicates & non-accepted words, computes `checkGuess`, updates game state.
    - `retryWordHandler`: restarts attempts on same `SessionWord`.
    - `gameStateHandler`: partial board render (HTMX).
    - `healthzHandler`: readiness/metrics.

## HTMX / Partial Rendering Pattern

Front-end uses `HX-Request` header to trigger partial template (`game-content`). Errors & events signaled via `HX-Trigger` header JSON / string (e.g. `rate-limit-exceeded`, `clear-completed-words`, server error codes). Preserve this behavior when adding interactive endpoints.

## Game Logic Pattern

`checkGuess(guess,target)` two-pass algorithm:

1. First pass marks exact matches and blanks those runes in a pooled buffer.
2. Second pass marks present/absent using remaining runes.
   Use existing `WordLength`, avoid allocating new rune slices unnecessarily (respect rune pool reuse). Always update via `updateGameState` to enforce win/lose transitions and set `TargetWord` only when `GameOver`.

## Sessions & Concurrency

-   Maps: `App.GameSessions`, `App.LimiterMap` protected by RWMutex. Follow existing pattern: read with `RLock`, then upgrade to `Lock` only when modifying. Avoid holding locks while rendering templates.
-   Cleanup routines (`startCleanupRoutines`) prune stale sessions & rate limiters using TTL env values (`SESSION_TTL`, `RATE_LIMITER_TTL`). If adding new maps, mirror this pattern for lifecycle management.

## Rate Limiting

Per client IP via `golang.org/x/time/rate`. Configuration from env: `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`. Mutating routes wrap with `app.rateLimitMiddleware()`. Emit `HX-Trigger` on limit exceed for HTMX clients.

## Security & Headers

-   CSP dynamically rewrites `'self'` per origin (`middleware.go`). Include new external domains explicitly if adding scripts/styles.
-   CSRF: cookie `csrf_token` must match header `X-CSRF-Token` or form field on mutating verbs (POST/PUT/PATCH/DELETE). Maintain this check for new write endpoints.
-   Production caching: static assets assigned `Cache-Control: public, max-age=<StaticCacheAge>`; dynamic routes: no-store. Respect `IsProduction` when adding new static mounts.

## Configuration (Environment)

Durations & integers resolved via `getEnvDuration` / `getEnvInt` with fallbacks:

-   `COOKIE_MAX_AGE`, `STATIC_CACHE_AGE`, `RATE_LIMITER_TTL`, `SESSION_TTL` etc.
    Use these helpers rather than manual parsing for consistency & logging.

## Testing Conventions

`go test ./...` – unit tests fabricate a minimal `App` via helper `testAppWithWords` rather than full server bootstrap. Emulate this for new logic: build narrow constructors with injected slices/maps. Focus on deterministic word lists.

## Adding Features – Preferred Patterns

-   Extend `App` for shared state; initialize in `main.go` before router setup.
-   New routes: register in `main.go`; use same middleware ordering.
-   Shared helper? Place in `util.go` or create new file; keep small & pure.
-   Error signaling to client: return error code constant; set `HX-Trigger` if HTMX.

## Gotchas / Edge Cases

-   All target words must be 5 letters; loader filters & warns. Ensure additions to `words.json` follow length & uppercase rules (internally stored uppercase for acceptance set).
-   Completed word exhaustion triggers `reset` flag (see `getRandomWordEntryExcluding`) – replicate if implementing multi-round features.
-   Avoid race: update `LastAccessTime` only while holding write lock.
-   `SessionWord` may be empty for legacy state; `getTargetWord` backfills safely.

## Quick Start for Agents

-   Run: `go run .` (or `air` for live reload).
-   Primary edit points: handlers (`handlers.go`), logic (`game.go`), middleware (`middleware.go`), templates (`templates/`).
-   Maintain HTMX headers & CSRF semantics when modifying forms.
