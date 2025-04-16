package collections

import (
	"gno.land/p/moul/md"

	"gno.land/p/lou/card"
)

func Render(_ string) string {
	out := md.H1("Collections") + "\n\n"
	out += md.HorizontalRule()
	out += "<gno-columns>\n\n"

	cards := []*card.Card{
		card.NewCard("Games", []string{
			"play.",
			"",
			"- /gno.land/demo/tamagotchi",
		}, 24, 0),

		card.NewCard("UI", []string{
			"markdown lets go.",
			"",
			"- /gno.land/p/moul/md",
			"- /gno.land/p/moul/mdtable",
		}, 24, 0),

		card.NewCard("Tech & Tools 🛠️", []string{
			"utilities and libraries.",
			"",
			"- /gno.land/p/demo/svg",
			"- /gno.land/p/moul/txlink",
		}, 24, 0),
	}

	for i, c := range cards {
		out += c.String()
		if i == 0 || i == 1 {
			out += "|||\n\n"
		}
	}
	out += "</gno-columns>\n\n"

	out += md.H2("Create & Share")
	out += "\n\n"
	out += "- Start your own collection.\n"
	out += "- Add packages from your realms.\n"
	out += "- Tag them for others to discover.\n"
	out += "- Join DAOs for collaborative collections.\n"

	return out
}
