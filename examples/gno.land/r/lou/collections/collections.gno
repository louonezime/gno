package collections

import (
	"gno.land/p/moul/md"

	"gno.land/p/lou/card"
)

// dao to create a shared collection???
// your own collection
// add tags

func Render(_ string) string {
	out := md.H1("Collections") + "\n\n"
	out += md.HorizontalRule()
	out += "<gno-columns>\n\n"

	cards := []*card.Card{
		card.NewCard("Philosophy 🧠", []string{
			"A space for deep thoughts.",
			"",
			"- /gno.land/p/socrates",
			"- /gno.land/p/zen",
			"- /gno.land/p/plato",
		}, 24, 0),

		card.NewCard("Memes 😂", []string{
			"Because we need humor.",
			"",
			"- /gno.land/p/wow",
			"- /gno.land/p/doge",
		}, 24, 0),

		card.NewCard("Tech & Tools 🛠️", []string{
			"Useful utilities and libraries.",
			"",
			"- /gno.land/p/ascii",
			"- /gno.land/p/mdtable",
			"- /gno.land/p/lou",
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
