package home

import "gno.land/r/leon/hor"

const staticHome = `# morgan's (gn)home

- [📝 sign my guestbook](/r/morgan/guestbook)
`

func init() { hor.Register("Morgan's Home Realm", "") }

func Render(path string) string {
	return staticHome
}
