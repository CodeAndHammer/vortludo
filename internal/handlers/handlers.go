package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/samber/lo"
	constants "vortludo/internal/constants"
	game "vortludo/internal/game"
	models "vortludo/internal/models"
	session "vortludo/internal/session"
	util "vortludo/internal/util"
)

func HomeHandler(app *models.App, c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := session.GetOrCreateSession(app, c)
	gameState := session.GetGameState(app, ctx, sessionID)
	hint := game.GetHintForWord(app, gameState.SessionWord)

	csrfToken, _ := c.Cookie("csrf_token")
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":      "Vortludo - A Libre Wordle Clone",
		"message":    "Guess the 5-letter word!",
		"hint":       hint,
		"game":       gameState,
		"csrf_token": csrfToken,
	})
}

func NewGameHandler(app *models.App, c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := session.GetOrCreateSession(app, c)
	util.LogInfo("Creating new game for session: %s", sessionID)

	var completedWords []string
	if c.Request.Method == "POST" {
		completedWordsStr := c.PostForm("completedWords")
		if completedWordsStr != "" {
			if err := json.Unmarshal([]byte(completedWordsStr), &completedWords); err != nil {
				util.LogWarn("Failed to parse completed words: %v", err)
				completedWords = []string{}
			} else {
				validCompletedWords := lo.Filter(completedWords, func(word string, _ int) bool {
					_, exists := app.WordSet[word]
					if !exists {
						util.LogWarn("Invalid completed word ignored: %s", word)
					}
					return exists
				})
				completedWords = validCompletedWords
				util.LogInfo("Validated %d completed words for session %s", len(completedWords), sessionID)
			}
		}
	}

	app.SessionMutex.Lock()
	delete(app.GameSessions, sessionID)
	app.SessionMutex.Unlock()
	util.LogInfo("Cleared old session data for: %s", sessionID)

	if c.Query("reset") == "1" {
		c.SetSameSite(http.SameSiteStrictMode)
		secure := app.IsProduction
		c.SetCookie(constants.SessionCookieName, "", -1, "/", "", secure, true)

		newSessionID := uuid.NewString()
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie(constants.SessionCookieName, newSessionID, int(app.CookieMaxAge.Seconds()), "/", "", secure, true)
		util.LogInfo("Created new session ID: %s", newSessionID)

		if len(completedWords) > 0 {
			_, needsReset := game.CreateNewGameWithCompletedWords(app, ctx, newSessionID, completedWords)
			if needsReset {
				c.Header("HX-Trigger", "clear-completed-words")
			}
		} else {
			game.CreateNewGame(app, ctx, newSessionID)
		}
		sessionID = newSessionID
	} else {
		if len(completedWords) > 0 {
			_, needsReset := game.CreateNewGameWithCompletedWords(app, ctx, sessionID, completedWords)
			if needsReset {
				c.Header("HX-Trigger", "clear-completed-words")
			}
		} else {
			game.CreateNewGame(app, ctx, sessionID)
		}
	}

	isHTMX := c.GetHeader("HX-Request") == "true"
	if isHTMX {
		gameState := session.GetGameState(app, ctx, sessionID)
		hint := game.GetHintForWord(app, gameState.SessionWord)
		csrfToken, _ := c.Cookie("csrf_token")
		c.HTML(http.StatusOK, "game-content", gin.H{
			"game":       gameState,
			"hint":       hint,
			"newGame":    true,
			"csrf_token": csrfToken,
		})
	} else {
		c.Redirect(http.StatusSeeOther, constants.RouteHome)
	}
}

func GuessHandler(app *models.App, c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := session.GetOrCreateSession(app, c)
	gameState := session.GetGameState(app, ctx, sessionID)
	hint := game.GetHintForWord(app, gameState.SessionWord)

	renderBoard := func(errCode string) {
		csrfToken, _ := c.Cookie("csrf_token")
		if errCode != "" {
			payload := map[string]string{"server_error_code": errCode}
			if b, jerr := json.Marshal(payload); jerr == nil {
				c.Header("HX-Trigger", string(b))
			} else {
				util.LogWarn("Failed to marshal HX-Trigger payload: %v", jerr)
			}
		}
		c.HTML(http.StatusOK, "game-content", gin.H{
			"game":       gameState,
			"hint":       hint,
			"error_code": errCode,
			"csrf_token": csrfToken,
		})
	}

	renderFullPage := func(errCode string) {
		csrfToken, _ := c.Cookie("csrf_token")
		if errCode != "" {
			payload := map[string]string{"server_error_code": errCode}
			if b, jerr := json.Marshal(payload); jerr == nil {
				c.Header("HX-Trigger", string(b))
			} else {
				util.LogWarn("Failed to marshal HX-Trigger payload: %v", jerr)
			}
		}
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":      "Vortludo - A Libre Wordle Clone",
			"message":    "Guess the 5-letter word!",
			"hint":       hint,
			"game":       gameState,
			"error_code": errCode,
			"csrf_token": csrfToken,
		})
	}

	isHTMX := c.GetHeader("HX-Request") == "true"
	var errCode string
	if err := ValidateGameState(app, c, gameState); err != nil {
		errCode = err.Error()
		if isHTMX {
			renderBoard(errCode)
		} else {
			renderFullPage(errCode)
		}
		return
	}

	guess := NormalizeGuess(c.PostForm("guess"))
	if !game.IsAcceptedWord(app, guess) {
		errCode = constants.ErrorCodeWordNotAccepted
		if isHTMX {
			renderBoard(errCode)
		} else {
			renderFullPage(errCode)
		}
		return
	}

	if slices.Contains(gameState.GuessHistory, guess) {
		errCode = constants.ErrorCodeDuplicateGuess
		if isHTMX {
			renderBoard(errCode)
		} else {
			renderFullPage(errCode)
		}
		return
	}
	if err := ProcessGuess(app, ctx, c, sessionID, gameState, guess, isHTMX, hint); err != nil {
		errCode = err.Error()
		if isHTMX {
			renderBoard(errCode)
		} else {
			renderFullPage(errCode)
		}
		return
	}
}

func GameStateHandler(app *models.App, c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := session.GetOrCreateSession(app, c)
	gameState := session.GetGameState(app, ctx, sessionID)
	hint := game.GetHintForWord(app, gameState.SessionWord)

	csrfToken, _ := c.Cookie("csrf_token")
	c.HTML(http.StatusOK, "game-content", gin.H{
		"game":       gameState,
		"hint":       hint,
		"csrf_token": csrfToken,
	})
}

func RetryWordHandler(app *models.App, c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := session.GetOrCreateSession(app, c)
	app.SessionMutex.Lock()
	gameState, exists := app.GameSessions[sessionID]
	if !exists {
		app.SessionMutex.Unlock()
		game.CreateNewGame(app, ctx, sessionID)
		c.Redirect(http.StatusSeeOther, "/")
		return
	}
	sessionWord := gameState.SessionWord
	guesses := lo.Times(constants.MaxGuesses, func(_ int) []models.GuessResult {
		return lo.Times(constants.WordLength, func(_ int) models.GuessResult { return models.GuessResult{} })
	})
	newGame := &models.GameState{
		Guesses:        guesses,
		CurrentRow:     0,
		GameOver:       false,
		Won:            false,
		TargetWord:     "",
		SessionWord:    sessionWord,
		GuessHistory:   []string{},
		LastAccessTime: time.Now(),
	}
	app.GameSessions[sessionID] = newGame
	app.SessionMutex.Unlock()
	c.Redirect(http.StatusSeeOther, "/")
}

func HealthzHandler(app *models.App, c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(app.StartTime)

	app.SessionMutex.RLock()
	sessionCount := len(app.GameSessions)
	app.SessionMutex.RUnlock()

	app.LimiterMutex.RLock()
	limiterCount := len(app.LimiterMap)
	app.LimiterMutex.RUnlock()

	c.JSON(http.StatusOK, gin.H{
		"status":          "ok",
		"env":             map[bool]string{true: "production", false: "development"}[app.IsProduction],
		"words_loaded":    len(app.WordList),
		"accepted_words":  len(app.AcceptedWordSet),
		"active_sessions": sessionCount,
		"active_limiters": limiterCount,
		"memory_alloc_mb": m.Alloc / 1024 / 1024,
		"memory_sys_mb":   m.Sys / 1024 / 1024,
		"memory_gc_count": m.NumGC,
		"uptime":          util.FormatUptime(uptime),
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	})
}

func ValidateGameState(app *models.App, _ *gin.Context, game *models.GameState) error {
	if game.GameOver {
		util.LogWarn("Session attempted guess on completed game")
		return errors.New(constants.ErrorCodeGameOver)
	}
	return nil
}

func NormalizeGuess(input string) string {
	return strings.ToUpper(strings.TrimSpace(input))
}

func ProcessGuess(app *models.App, ctx context.Context, c *gin.Context, sessionID string, gameState *models.GameState, guess string, isHTMX bool, hint string) error {
	util.LogInfo("Session %s guessed: %s (attempt %d/%d)", sessionID, guess, gameState.CurrentRow+1, constants.MaxGuesses)

	if len(guess) != constants.WordLength {
		util.LogWarn("Session %s submitted invalid length guess: %s (%d letters)", sessionID, guess, len(guess))
		return errors.New(constants.ErrorCodeInvalidLength)
	}

	if gameState.CurrentRow >= constants.MaxGuesses {
		util.LogWarn("Session %s attempted guess after max guesses reached", sessionID)
		return errors.New(constants.ErrorCodeNoMoreGuesses)
	}

	targetWord := game.GetTargetWord(app, ctx, gameState)
	isInvalid := !game.IsValidWord(app, guess)
	result := game.CheckGuess(guess, targetWord, app)
	game.UpdateGameState(app, ctx, gameState, guess, targetWord, result, isInvalid)
	session.SaveGameState(app, sessionID, gameState)

	if isHTMX {
		c.HTML(http.StatusOK, "game-content", gin.H{"game": gameState, "hint": hint})
	} else {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title":   "Vortludo - A Libre Wordle Clone",
			"message": "Guess the 5-letter word!",
			"hint":    hint,
			"game":    gameState,
		})
	}
	return nil
}
