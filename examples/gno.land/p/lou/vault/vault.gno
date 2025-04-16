package vault

import (
	"std"

	"gno.land/p/demo/grc/grc20"
)

type Deposit struct {
	Amount uint64
	Owner  std.Address
	Unlock int64 // block height or timestamp
}

var deposits = map[string]Deposit{}
var token grc20.Token
var ledger grc20.PrivateLedger

func Init(tok grc20.Token) {
	token = tok
}

// Deposit tokens to vault
func Lock(id string, amount uint64, unlock int64) {
	from := std.PreviousRealm().Address()
	if err := ledger.Transfer(from, std.PreviousRealm().Address(), amount); err != nil {
		panic(err)
	}
	deposits[id] = Deposit{
		Amount: amount,
		Owner:  from,
		Unlock: unlock,
	}
}

// Withdraw if time passed
func Withdraw(id string) {
	now := std.ChainHeight()
	dep := deposits[id]

	if dep.Owner != std.PreviousRealm().Address() {
		panic("not owner")
	}
	if now < dep.Unlock {
		panic("vault still locked")
	}

	delete(deposits, id)
	ledger.Transfer(std.PreviousRealm().Address(), dep.Owner, dep.Amount)
}
