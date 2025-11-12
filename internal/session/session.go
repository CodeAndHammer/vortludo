package session

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	constants "vortludo/internal/constants"
	game "vortludo/internal/game"
	models "vortludo/internal/models"
	util "vortludo/internal/util"
)

func GetOrCreateSession(app *models.App, c *gin.Context) string {
	sessionID, err := c.Cookie(constants.SessionCookieName)
	if err != nil || len(sessionID) < 10 {
		sessionID = uuid.NewString()
		c.SetSameSite(http.SameSiteStrictMode)
		secure := app.IsProduction
		c.SetCookie(constants.SessionCookieName, sessionID, int(app.CookieMaxAge.Seconds()), "/", "", secure, true)
		util.LogInfo("Created new session: %s", sessionID)
	}
	return sessionID
}

func GetGameState(app *models.App, ctx context.Context, sessionID string) *models.GameState {
	app.SessionMutex.RLock()
	gameState, exists := app.GameSessions[sessionID]
	app.SessionMutex.RUnlock()
	if exists {
		app.SessionMutex.Lock()
		gameState.LastAccessTime = time.Now()
		app.SessionMutex.Unlock()
		util.LogInfo("Retrieved cached game state for session: %s, updated last access time.", sessionID)
		return gameState
	}

	util.LogInfo("Creating new game for session: %s", sessionID)
	return game.CreateNewGame(app, ctx, sessionID)
}

func SaveGameState(app *models.App, sessionID string, game *models.GameState) {
	app.SessionMutex.Lock()
	app.GameSessions[sessionID] = game
	game.LastAccessTime = time.Now()
	app.SessionMutex.Unlock()
	util.LogInfo("Updated in-memory game state for session: %s", sessionID)
}

func CleanupExpiredSessions(app *models.App) {
	app.SessionMutex.Lock()
	defer app.SessionMutex.Unlock()

	now := time.Now()
	expiredCount := 0
	for sessionID, game := range app.GameSessions {
		if now.Sub(game.LastAccessTime) > app.SessionTimeout {
			delete(app.GameSessions, sessionID)
			expiredCount++
		}
	}

	if expiredCount > 0 {
		util.LogInfo("Cleaned up %d expired sessions", expiredCount)
	}
}

func StartSessionCleanup(app *models.App) {
	ticker := time.NewTicker(10 * time.Minute)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			CleanupExpiredSessions(app)
		}
	}()
	util.LogInfo("Started session cleanup goroutine")
}
