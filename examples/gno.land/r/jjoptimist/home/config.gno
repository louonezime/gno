package home

import (
	"std"

	"gno.land/p/demo/ownable"
)

type Config struct {
	Title       string
	Description string
	Github      string
}

var config = Config{
	Title:       "JJOptimist's Home Realm 🏠",
	Description: "Exploring Gno and building on-chain",
	Github:      "jjoptimist",
}

var Ownable = ownable.NewWithAddress(std.Address("g16vfw3r7zuz43fhky3xfsuc2hdv9tnhvlkyn0nj"))

func GetConfig() Config {
	return config
}

func UpdateConfig(newTitle, newDescription, newGithub string) {
	Ownable.AssertOwnedByPrevious()
	config.Title = newTitle
	config.Description = newDescription
	config.Github = newGithub
}
