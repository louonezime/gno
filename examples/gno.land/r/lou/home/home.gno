package home

import (
	"gno.land/p/moul/md"
)

func Render(path string) string {
	out := md.H1("Lou's Home\n\n")
	out += md.HorizontalRule()

	out += RenderRockPaperScissors()
	out += RenderMagic8Ball()
	out += RenderRatings()

	return out
}
