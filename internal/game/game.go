package game

import (
	"context"
	"crypto/rand"
	"math/big"
	"slices"
	"time"

	"github.com/samber/lo"
	constants "vortludo/internal/constants"
	models "vortludo/internal/models"
	util "vortludo/internal/util"
)

func GetRandomWordEntry(app *models.App, ctx context.Context) models.WordEntry {
	reqID, _ := ctx.Value(constants.RequestIDKey).(string)

	select {
	case <-ctx.Done():
		if reqID != "" {
			util.LogWarn("[request_id=%v] GetRandomWordEntry cancelled: %v", reqID, ctx.Err())
		} else {
			util.LogWarn("GetRandomWordEntry cancelled: %v", ctx.Err())
		}
		return app.WordList[0]
	default:
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(app.WordList))))
	if err != nil {
		if reqID != "" {
			util.LogWarn("[request_id=%v] Error generating random number: %v, using fallback", reqID, err)
		} else {
			util.LogWarn("Error generating random number: %v, using fallback", err)
		}
		return app.WordList[0]
	}

	if reqID != "" {
		util.LogInfo("[request_id=%v] Selected random word index: %d", reqID, n.Int64())
	}
	return app.WordList[n.Int64()]
}

func GetRandomWordEntryExcluding(app *models.App, ctx context.Context, completedWords []string) (models.WordEntry, bool) {
	reqID, _ := ctx.Value(constants.RequestIDKey).(string)

	if len(completedWords) == 0 {
		return GetRandomWordEntry(app, ctx), false
	}

	availableWords := lo.Filter(app.WordList, func(entry models.WordEntry, _ int) bool {
		return !slices.Contains(completedWords, entry.Word)
	})

	if len(availableWords) == 0 {
		if reqID != "" {
			util.LogInfo("[request_id=%v] All words completed, reset needed. Total words: %d, Completed: %d", reqID, len(app.WordList), len(completedWords))
		} else {
			util.LogInfo("All words completed, reset needed. Total words: %d, Completed: %d", len(app.WordList), len(completedWords))
		}
		return GetRandomWordEntry(app, ctx), true
	}

	select {
	case <-ctx.Done():
		if reqID != "" {
			util.LogWarn("[request_id=%v] GetRandomWordEntryExcluding cancelled: %v", reqID, ctx.Err())
		} else {
			util.LogWarn("GetRandomWordEntryExcluding cancelled: %v", ctx.Err())
		}
		return availableWords[0], false
	default:
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(availableWords))))
	if err != nil {
		if reqID != "" {
			util.LogWarn("[request_id=%v] Error generating random number for filtered words: %v, using fallback", reqID, err)
		} else {
			util.LogWarn("Error generating random number for filtered words: %v, using fallback", err)
		}
		return availableWords[0], false
	}

	selected := availableWords[n.Int64()]
	if reqID != "" {
		util.LogInfo("[request_id=%v] Selected word from %d available options (excluding %d completed): %s", reqID, len(availableWords), len(completedWords), selected.Word)
	} else {
		util.LogInfo("Selected word from %d available options (excluding %d completed): %s", len(availableWords), len(completedWords), selected.Word)
	}

	return selected, false
}

func GetHintForWord(app *models.App, wordValue string) string {
	if wordValue == "" {
		return ""
	}
	hint, ok := app.HintMap[wordValue]
	if ok {
		return hint
	}
	util.LogWarn("Hint not found for word: %s", wordValue)
	return ""
}

func BuildHintMap(wordList []models.WordEntry) map[string]string {
	return lo.Associate(wordList, func(entry models.WordEntry) (string, string) {
		return entry.Word, entry.Hint
	})
}

func GetTargetWord(app *models.App, ctx context.Context, game *models.GameState) string {
	if game.SessionWord == "" {
		selectedEntry := GetRandomWordEntry(app, ctx)
		game.SessionWord = selectedEntry.Word
		util.LogWarn("SessionWord was empty, assigned random word: %s", selectedEntry.Word)
	}
	return game.SessionWord
}

func UpdateGameState(app *models.App, ctx context.Context, game *models.GameState, guess, targetWord string, result []models.GuessResult, isInvalid bool) {
	reqID, _ := ctx.Value(constants.RequestIDKey).(string)

	if game.CurrentRow >= constants.MaxGuesses {
		return
	}

	game.Guesses[game.CurrentRow] = result
	game.GuessHistory = append(game.GuessHistory, guess)
	game.LastAccessTime = time.Now()

	if !isInvalid && guess == targetWord {
		game.Won = true
		game.GameOver = true
		if reqID != "" {
			util.LogInfo("[request_id=%v] Player won! Target word was: %s", reqID, targetWord)
		} else {
			util.LogInfo("Player won! Target word was: %s", targetWord)
		}
	} else {
		game.CurrentRow++

		if game.CurrentRow >= constants.MaxGuesses {
			game.GameOver = true
			if reqID != "" {
				util.LogInfo("[request_id=%v] Player lost. Target word was: %s", reqID, targetWord)
			} else {
				util.LogInfo("Player lost. Target word was: %s", targetWord)
			}
		}
	}

	if game.GameOver {
		game.TargetWord = targetWord
	}
}

func CheckGuess(guess, target string, app *models.App) []models.GuessResult {
	result := make([]models.GuessResult, constants.WordLength)
	var targetCopy []rune
	var pooledBuf []rune
	usedPool := false

	if app.RuneBufPool != nil {
		if v := app.RuneBufPool.Get(); v != nil {
			if ptr, ok := v.(*[]rune); ok && ptr != nil {
				pooledBuf = *ptr
				targetCopy = pooledBuf[:constants.WordLength]
				copy(targetCopy, []rune(target))
				usedPool = true
			} else {
				targetCopy = []rune(target)
			}
		} else {
			targetCopy = []rune(target)
		}
	} else {
		targetCopy = []rune(target)
	}

	for i := range constants.WordLength {
		if guess[i] == target[i] {
			result[i] = models.GuessResult{Letter: string(guess[i]), Status: constants.GuessStatusCorrect}
			targetCopy[i] = ' '
		}
	}

	for i := range constants.WordLength {
		if result[i].Status == "" {
			letter := string(guess[i])
			result[i].Letter = letter

			found := false
			for j := range constants.WordLength {
				if targetCopy[j] == rune(guess[i]) {
					result[i].Status = constants.GuessStatusPresent
					targetCopy[j] = ' '
					found = true
					break
				}
			}

			if !found {
				result[i].Status = constants.GuessStatusAbsent
			}
		}
	}

	if usedPool {
		for i := range pooledBuf {
			pooledBuf[i] = 0
		}
		if app.RuneBufPool != nil {
			buf := pooledBuf
			app.RuneBufPool.Put(&buf)
		}
	}

	return result
}

func IsValidWord(app *models.App, word string) bool {
	_, ok := app.WordSet[word]
	return ok
}

func IsAcceptedWord(app *models.App, word string) bool {
	_, ok := app.AcceptedWordSet[word]
	return ok
}

func CreateNewGame(app *models.App, ctx context.Context, sessionID string) *models.GameState {
	selectedEntry := GetRandomWordEntry(app, ctx)
	util.LogInfo("New game created for session %s with word: %s (hint: %s)", sessionID, selectedEntry.Word, selectedEntry.Hint)
	guesses := lo.Times(constants.MaxGuesses, func(_ int) []models.GuessResult {
		return lo.Times(constants.WordLength, func(_ int) models.GuessResult { return models.GuessResult{} })
	})
	game := &models.GameState{
		Guesses:        guesses,
		CurrentRow:     0,
		GameOver:       false,
		Won:            false,
		TargetWord:     "",
		SessionWord:    selectedEntry.Word,
		GuessHistory:   []string{},
		LastAccessTime: time.Now(),
	}
	app.GameSessions[sessionID] = game
	return game
}

func CreateNewGameWithCompletedWords(app *models.App, ctx context.Context, sessionID string, completedWords []string) (*models.GameState, bool) {
	selectedEntry, needsReset := GetRandomWordEntryExcluding(app, ctx, completedWords)
	util.LogInfo("New game created for session %s with word: %s (hint: %s, completed words: %d, needs reset: %v)",
		sessionID, selectedEntry.Word, selectedEntry.Hint, len(completedWords), needsReset)

	guesses := lo.Times(constants.MaxGuesses, func(_ int) []models.GuessResult {
		return lo.Times(constants.WordLength, func(_ int) models.GuessResult { return models.GuessResult{} })
	})
	game := &models.GameState{
		Guesses:        guesses,
		CurrentRow:     0,
		GameOver:       false,
		Won:            false,
		TargetWord:     "",
		SessionWord:    selectedEntry.Word,
		GuessHistory:   []string{},
		LastAccessTime: time.Now(),
	}
	app.GameSessions[sessionID] = game
	return game, needsReset
}
