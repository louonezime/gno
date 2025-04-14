package home

import (
	"std"

	"gno.land/p/moul/md"

	"gno.land/p/lou/tokenaccess"
)

func Render(path string) string {
	out := "<gno-columns>\n\n"
	out += md.Image("Github profile picture", "https://avatars.githubusercontent.com/u/109749395?v=4&width=30") + "\n\n"
	out += "|||\n\n"

	out += "# Lou's Home\n"
	out += md.HorizontalRule()
	out += RenderRockPaperScissors()
	if err := tokenaccess.RequireMinimumAllowance(token, admin, 10, std.Address(currentProfile.Account())); err != nil {
		out += md.H2("No further content unlocked.") + "\n\n"
		out += md.Italic("--> "+err.Error()) + "\n\n"
	} else {
		out += RenderMagic8Ball()
	}
	out += "</gno-columns>\n\n"

	out += RenderRatings()
	return out
}
