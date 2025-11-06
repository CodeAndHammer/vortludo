package main

const (
	MaxGuesses = 6
	WordLength = 5
)

const (
	GuessStatusCorrect = "correct"
	GuessStatusPresent = "present"
	GuessStatusAbsent  = "absent"
)

const (
	SessionCookieName = "session_id"
)

const (
	RouteHome      = "/"
	RouteNewGame   = "/new-game"
	RouteRetryWord = "/retry-word"
	RouteGuess     = "/guess"
	RouteGameState = "/game-state"
)

const (
	ErrorCodeGameOver        = "game_over"
	ErrorCodeInvalidLength   = "invalid_length"
	ErrorCodeNoMoreGuesses   = "no_more_guesses"
	ErrorCodeNotInWordList   = "not_in_word_list"
	ErrorCodeWordNotAccepted = "word_not_accepted"
	ErrorCodeDuplicateGuess  = "duplicate_guess"
)

const (
	requestIDKey contextKey = "request_id"
)
