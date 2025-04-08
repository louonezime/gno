package eightball

import (
	"std"
)

var (
	decisions = []string{
		"It is certain.",
		"It is decidedly so.",
		"Without a doubt.",
		"Yes definitely.",
		"You may rely on it.",
		"As I see it, yes.",
		"Most likely.",
		"Outlook looks good.",
		"Yes.",
		"Signs point to yes.",
		"Reply hazy, try again.",
		"Ask again later.",
		"Better not tell you now.",
		"Cannot predict now.",
		"Concentrate and ask again.",
		"Don't count on it.",
		"My reply is no.",
		"My sources say no.",
		"Outlook not so good.",
		"Very doubtful.",
	}
)

func GetResponse(seed string) string {
	index := (HashString(seed) + int(std.ChainHeight())%len(decisions))
	return decisions[index]
}

func HashString(s string) int {
	const prime = 31
	hash := 0

	for i := 0; i < len(s); i++ {
		hash = hash*prime + int(s[i])
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}
