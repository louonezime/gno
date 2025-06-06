// Package bar20 is similar to gno.land/r/demo/foo20 but exposes a safe-object
// that can be used by `maketx run`, another contract importing foo20, and in
// the future when we'll support `maketx call Token.XXX`.
package bar20

import (
	"std"
	"strings"

	"gno.land/p/demo/grc/grc20"
	"gno.land/p/demo/ufmt"
	"gno.land/r/demo/grc20reg"
)

var (
	Token, adm = grc20.NewToken("Bar", "BAR", 4)
	UserTeller = Token.CallerTeller()
)

func init() {
	cross(grc20reg.Register)(Token.Getter(), "")
}

func Faucet() string {
	crossing()

	caller := std.PreviousRealm().Address()
	if err := adm.Mint(caller, 1_000_000); err != nil {
		return "error: " + err.Error()
	}
	return "OK"
}

func Render(path string) string {
	parts := strings.Split(path, "/")
	c := len(parts)

	switch {
	case path == "":
		return Token.RenderHome()
	case c == 2 && parts[0] == "balance":
		owner := std.Address(parts[1])
		balance := Token.BalanceOf(owner)
		return ufmt.Sprintf("%d\n", balance)
	default:
		return "404\n"
	}
}
