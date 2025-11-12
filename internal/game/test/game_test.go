package main

import (
	"context"
	"testing"

	constants "vortludo/internal/constants"
	game "vortludo/internal/game"
	models "vortludo/internal/models"
)

func testAppWithWords(words []models.WordEntry) *models.App {
	wordSet := make(map[string]struct{})
	acceptedSet := make(map[string]struct{})
	hintMap := make(map[string]string)
	for _, w := range words {
		wordSet[w.Word] = struct{}{}
		acceptedSet[w.Word] = struct{}{}
		hintMap[w.Word] = w.Hint
	}
	return &models.App{
		WordList:        words,
		WordSet:         wordSet,
		AcceptedWordSet: acceptedSet,
		HintMap:         hintMap,
		GameSessions:    make(map[string]*models.GameState),
	}
}

func dummyContext() context.Context {
	return context.Background()
}

func TestGetRandomWordEntry(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}, {Word: "table", Hint: "furniture"}}
	app := testAppWithWords(words)
	ctx := dummyContext()
	found := false
	for i := 0; i < 10; i++ {
		w := game.GetRandomWordEntry(app, ctx)
		if w.Word == "apple" || w.Word == "table" {
			found = true
		} else {
			t.Errorf("Unexpected word: %v", w.Word)
		}
	}
	if !found {
		t.Error("getRandomWordEntry did not return any valid word")
	}
}

func TestGetRandomWordEntryExcluding(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}, {Word: "table", Hint: "furniture"}}
	app := testAppWithWords(words)
	ctx := dummyContext()
	w, reset := game.GetRandomWordEntryExcluding(app, ctx, []string{"apple"})
	if w.Word != "table" || reset {
		t.Errorf("Expected table, got %v, reset=%v", w.Word, reset)
	}
	w, reset = game.GetRandomWordEntryExcluding(app, ctx, []string{"apple", "table"})
	if reset != true {
		t.Error("Expected reset=true when all words completed")
	}
}

func TestGetHintForWord(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}}
	app := testAppWithWords(words)
	if game.GetHintForWord(app, "apple") != "fruit" {
		t.Error("Expected hint 'fruit'")
	}
	if game.GetHintForWord(app, "") != "" {
		t.Error("Expected empty string for empty word")
	}
	if game.GetHintForWord(app, "unknown") != "" {
		t.Error("Expected empty string for unknown word")
	}
}

func TestBuildHintMap(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}, {Word: "table", Hint: "furniture"}}
	hm := game.BuildHintMap(words)
	if hm["apple"] != "fruit" || hm["table"] != "furniture" {
		t.Error("Hint map not built correctly")
	}
}

func TestGetTargetWord(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}}
	app := testAppWithWords(words)
	ctx := dummyContext()
	gameState := &models.GameState{SessionWord: ""}
	w := game.GetTargetWord(app, ctx, gameState)
	if w != "apple" {
		t.Errorf("Expected 'apple', got %v", w)
	}
	gameState.SessionWord = "table"
	w = game.GetTargetWord(app, ctx, gameState)
	if w != "table" {
		t.Errorf("Expected 'table', got %v", w)
	}
}

func TestUpdateGameState_WinLose(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}}
	app := testAppWithWords(words)
	ctx := dummyContext()
	gameState := &models.GameState{
		Guesses:      make([][]models.GuessResult, constants.MaxGuesses),
		CurrentRow:   0,
		GameOver:     false,
		Won:          false,
		TargetWord:   "",
		SessionWord:  "apple",
		GuessHistory: []string{},
	}
	result := []models.GuessResult{{Letter: "a", Status: constants.GuessStatusCorrect}, {Letter: "p", Status: constants.GuessStatusCorrect}, {Letter: "p", Status: constants.GuessStatusCorrect}, {Letter: "l", Status: constants.GuessStatusCorrect}, {Letter: "e", Status: constants.GuessStatusCorrect}}
	game.UpdateGameState(app, ctx, gameState, "apple", "apple", result, false)
	if !gameState.Won || !gameState.GameOver || gameState.TargetWord != "apple" {
		t.Error("Game should be won and over, target word revealed")
	}

	gameState = &models.GameState{
		Guesses:      make([][]models.GuessResult, constants.MaxGuesses),
		CurrentRow:   constants.MaxGuesses - 1,
		GameOver:     false,
		Won:          false,
		TargetWord:   "",
		SessionWord:  "apple",
		GuessHistory: []string{},
	}
	game.UpdateGameState(app, ctx, gameState, "wrong", "apple", result, false)
	if !gameState.GameOver || gameState.Won {
		t.Error("Game should be over and lost")
	}
	if gameState.TargetWord != "apple" {
		t.Error("Target word should be revealed on loss")
	}
}

func TestCheckGuess(t *testing.T) {
	res := game.CheckGuess("apple", "apple", &models.App{})
	for _, r := range res {
		if r.Status != constants.GuessStatusCorrect {
			t.Error("All should be correct")
		}
	}

	res = game.CheckGuess("zzzzz", "apple", &models.App{})
	for _, r := range res {
		if r.Status != constants.GuessStatusAbsent {
			t.Error("All should be absent")
		}
	}

	res = game.CheckGuess("pleap", "apple", &models.App{})
	statuses := []string{constants.GuessStatusPresent, constants.GuessStatusPresent, constants.GuessStatusPresent, constants.GuessStatusPresent, constants.GuessStatusPresent}
	for i, r := range res {
		if r.Status != statuses[i] {
			t.Errorf("Expected %v at %d, got %v", statuses[i], i, r.Status)
		}
	}
}

func TestIsValidWordAndIsAcceptedWord(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}}
	app := testAppWithWords(words)
	if !game.IsValidWord(app, "apple") {
		t.Error("apple should be valid")
	}
	if game.IsValidWord(app, "table") {
		t.Error("table should not be valid")
	}
	if !game.IsAcceptedWord(app, "apple") {
		t.Error("apple should be accepted")
	}
	if game.IsAcceptedWord(app, "table") {
		t.Error("table should not be accepted")
	}
}

func TestCreateNewGame(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}}
	app := testAppWithWords(words)
	ctx := dummyContext()
	gameState := game.CreateNewGame(app, ctx, "sess1")
	if gameState.SessionWord != "apple" {
		t.Error("SessionWord should be 'apple'")
	}
	if len(gameState.Guesses) != constants.MaxGuesses {
		t.Error("Guesses length incorrect")
	}
	if app.GameSessions["sess1"] == nil {
		t.Error("Game not stored in session map")
	}
}

func TestCreateNewGameWithCompletedWords(t *testing.T) {
	words := []models.WordEntry{{Word: "apple", Hint: "fruit"}, {Word: "table", Hint: "furniture"}}
	app := testAppWithWords(words)
	ctx := dummyContext()
	gameState, reset := game.CreateNewGameWithCompletedWords(app, ctx, "sess2", []string{"apple"})
	if gameState.SessionWord != "table" || reset {
		t.Error("Should select 'table' and reset=false")
	}
	_, reset = game.CreateNewGameWithCompletedWords(app, ctx, "sess3", []string{"apple", "table"})
	if !reset {
		t.Error("Should set reset=true when all words completed")
	}
}
