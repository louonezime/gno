package rps

import (
	"errors"
	"std"

	"gno.land/p/demo/ufmt"

	"gno.land/p/lou/ascii"
)

const (
	Rock     = "rock"
	Paper    = "paper"
	Scissors = "scissors"
)

var (
	hands = []string{
		Rock,
		Paper,
		Scissors,
	}
	winMap = map[string]string{
		Rock:     Scissors,
		Paper:    Rock,
		Scissors: Paper,
	}

	ErrInvalidHand = errors.New("Invalid hand input: only rock, paper or scissors")
)

type Profile struct {
	wins    int64
	losses  int64
	account string
}

type GameSession struct {
	hasWon bool
	isATie bool
	detail string
}

func (profile Profile) Wins() int64 {
	return profile.wins
}

func (profile Profile) Losses() int64 {
	return profile.losses
}

func (profile Profile) Account() string {
	return profile.account
}

func (profile *Profile) SetWins(wins int64) {
	profile.wins = wins
}

func (profile *Profile) SetLosses(losses int64) {
	profile.losses = losses
}

func NewProfile(account string) *Profile {
	return &Profile{
		wins:    0,
		losses:  0,
		account: account,
	}
}

func NewSession() *GameSession {
	return &GameSession{
		hasWon: false,
		isATie: false,
		detail: "",
	}
}

func (game *GameSession) IsATie() bool {
	return game.isATie
}

func (game *GameSession) HasWon() bool {
	return game.hasWon
}

func (game *GameSession) SendHand(input string) error {
	if _, ok := winMap[input]; !ok {
		return ErrInvalidHand
	}

	opponent := GetHand()
	game.isATie = false
	game.hasWon = false

	if input == opponent {
		game.detail = "It's a **tie**! Try Again!\n"
		game.isATie = true
	} else if winMap[input] == opponent {
		game.detail = ufmt.Sprintf("**%s** beats **%s**! You Won!\n", input, opponent)
		game.hasWon = true
	} else {
		game.detail = ufmt.Sprintf("**%s** beats **%s**! You Lost!\n", opponent, input)
	}
	return nil
}

func GetHand() string {
	random := int(std.ChainHeight()) % len(hands)
	return hands[random]
}

func (profile Profile) String() string {
	out := ""

	if profile.Wins() > 0 || profile.Losses() > 0 {
		out += "Career: "
	}
	out += ascii.Repeat("🏆", int(profile.Wins()))
	out += ascii.Repeat("❌", int(profile.Losses()))
	return out
}

func (game GameSession) String() string {
	return game.detail
}
