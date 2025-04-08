package comment

import (
	"std"

	"gno.land/p/demo/seqid"
)

type Comment struct {
	rating  uint64
	comment *string
	author  std.Address
	id      seqid.ID
}

func (c Comment) Rating() uint64 {
	return c.rating
}

func (c Comment) Comment() *string {
	return c.comment
}

func (c Comment) Author() std.Address {
	return c.author
}

func (c Comment) ID() seqid.ID {
	return c.id
}

func NewComment(rating uint64, comment string, author std.Address, id seqid.ID) *Comment {
	return &Comment{
		rating:  rating,
		comment: &comment,
		author:  author,
		id:      id,
	}
}

func NewRating(rating uint64, author std.Address, id seqid.ID) *Comment {
	return &Comment{
		rating:  rating,
		comment: nil,
		author:  author,
		id:      id,
	}
}
