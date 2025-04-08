package home

import (
	"std"

	"gno.land/p/demo/avl"
	"gno.land/p/moul/md"
	"gno.land/p/moul/txlink"

	"gno.land/p/lou/eightball"
	"gno.land/p/lou/rps"
)

var (
	profiles        *avl.Tree
	currentProfile  *rps.Profile
	gameSession     *rps.GameSession
	eightballAnswer string
)

func init() {
	currentProfile = rps.NewProfile("")
	gameSession = rps.NewSession()
	profiles = avl.NewTree()
}

// ROCK PAPER SCISSORS

func Play(input string) {
	caller := std.PreviousRealm().Address()
	addr := caller.String()

	value, ok := profiles.Get(addr)
	var profile *rps.Profile

	if ok {
		profile, _ = value.(*rps.Profile)
	}
	if profile == nil {
		profile = rps.NewProfile("")
		profiles.Set(addr, profile)
	}
	currentProfile = profile

	if err := gameSession.SendHand(input); err != nil {
		panic(err)
	}

	if gameSession.IsATie() {
		return
	}

	if gameSession.HasWon() {
		profile.SetWins(profile.Wins() + 1)
	} else {
		profile.SetLosses(profile.Losses() + 1)
	}
}

func RenderRockPaperScissors() string {
	out := md.H2("Rock, Paper, Scissor\n\n")
	out += currentProfile.String()
	out += "\n\nPlay [ "
	out += md.Link("Rock", txlink.Call("Play", "input", "rock"))
	out += " | "
	out += md.Link("Paper", txlink.Call("Play", "input", "paper"))
	out += " | "
	out += md.Link("Scissors ", txlink.Call("Play", "input", "scissors"))
	out += " ]\n\n" + gameSession.String()
	out += "\n\n"
	return out
}

// MAGIC 8 BALL

func Ask(question string) string {
	answer := eightball.GetResponse(question)
	eightballAnswer = answer + "\n\n"
	return answer
}

func RenderMagic8Ball() string {
	out := md.H2("Magic 8 Ball\n\n")
	out += md.Link("[ ASK A QUESTION ]", txlink.Call("Ask", "question", "Question"))
	out += "\n\n"
	out += eightballAnswer
	return out
}
