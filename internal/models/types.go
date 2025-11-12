package models

import (
	"sync"
	"time"
)

type WordEntry struct {
	Word string `json:"word"`
	Hint string `json:"hint"`
}

type WordList struct {
	Words []WordEntry `json:"words"`
}

type GameState struct {
	Guesses        [][]GuessResult `json:"guesses"`
	CurrentRow     int             `json:"currentRow"`
	GameOver       bool            `json:"gameOver"`
	Won            bool            `json:"won"`
	TargetWord     string          `json:"targetWord"`
	SessionWord    string          `json:"sessionWord"`
	GuessHistory   []string        `json:"guessHistory"`
	LastAccessTime time.Time       `json:"lastAccessTime"`
}

type GuessResult struct {
	Letter string `json:"letter"`
	Status string `json:"status"`
}

// rateLimiterEntry represents a rate limiter entry for a client IP
type RateLimiterEntry struct {
	Limiter        interface{} // would be golang.org/x/time/rate.Limiter in actual usage
	LastAccessTime time.Time
}

type App struct {
	WordList        []WordEntry
	WordSet         map[string]struct{}
	AcceptedWordSet map[string]struct{}
	HintMap         map[string]string
	GameSessions    map[string]*GameState
	SessionMutex    sync.RWMutex
	LimiterMap      map[string]*RateLimiterEntry
	LimiterMutex    sync.RWMutex
	IsProduction    bool
	StartTime       time.Time
	CookieMaxAge    time.Duration
	StaticCacheAge  time.Duration
	RateLimitRPS    int
	RateLimitBurst  int
	SessionTimeout  time.Duration
	RuneBufPool     *sync.Pool
}
