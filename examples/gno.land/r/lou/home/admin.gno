package home

import (
	"gno.land/p/demo/avl"
	"gno.land/p/demo/ownable"
	"gno.land/p/demo/seqid"
)

var Ownable = ownable.NewWithAddress("g14x2crt492f84shxa9s7e5ulk7lnf2p3euz7r9n")

func ResetRatings() {
	Ownable.AssertCallerIsOwner()
	ratings = avl.NewTree()
	ratingID = seqid.ID(0)
}
