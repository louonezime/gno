package home

import (
	"std"

	"gno.land/p/demo/avl"
	"gno.land/p/demo/seqid"
	"gno.land/p/demo/ufmt"
	"gno.land/p/moul/md"
	"gno.land/p/moul/txlink"
	"gno.land/r/sys/users"

	"gno.land/p/lou/comment"
)

var (
	ratings  *avl.Tree
	ratingID seqid.ID
)

func init() {
	ratings = avl.NewTree()
	ratingID = seqid.ID(0)
}

func NewRating(rating uint64) string {
	return addRating(rating, "")
}

func NewRatingComment(rating uint64, comment string) string {
	return addRating(rating, comment)
}

func addRating(rating uint64, commment string) string {
	caller := std.PreviousRealm().Address()
	if _, exists := ratings.Get(caller.String()); exists {
		panic(caller.String() + " has already rated and update instead")
	}
	id := ratingID.Next()

	var p *comment.Comment
	if commment == "" {
		p = comment.NewRating(rating, std.PreviousRealm().Address(), id)
	} else {
		p = comment.NewComment(rating, commment, std.PreviousRealm().Address(), id)
	}

	ratings.Set(caller.String(), p)
	return ufmt.Sprintf("Successfully added a #%d rating!", rating)
}

func UpdateRating(rating uint64) string {
	return updateRating(rating, "")
}

func UpdateRatingComment(rating uint64, comment string) string {
	return updateRating(rating, comment)
}

func updateRating(rating uint64, commentStr string) string {
	checkoutCaller()
	caller := std.PreviousRealm().Address()

	val, removed := ratings.Remove(caller.String())
	oldComment := val.(*comment.Comment)
	if !removed {
		panic("Removal of rating was unsuccessful")
	}
	if commentStr == "" && oldComment.Comment() != nil {
		commentStr = *oldComment.Comment()
	}
	return addRating(rating, commentStr) + "\nSuccessfully updated rating!\n"
}

func UpdateByID(id seqid.ID, rating uint64, comment string) string {
	caller := std.PreviousRealm().Address()

	if key, found := findRatingByID(id); found {
		if key == caller.String() {
			return updateRating(rating, comment)
		}
	}
	panic("Not authorized to update this rating")
}

func DeleteRating(id seqid.ID) string {
	caller := std.PreviousRealm().Address()

	if Ownable.CallerIsOwner() {
		if key, found := findRatingByID(id); found {
			ratings.Remove(key)
		}
		return ufmt.Sprintf("Rating as admin was removed successfully!")
	}
	checkoutCaller()
	if _, removed := ratings.Remove(caller.String()); !removed {
		panic("Invalid removal of comment")
	}
	return ufmt.Sprintf("Rating was removed successfully!")
}

func checkoutCaller() {
	caller := std.PreviousRealm().Address()

	if _, exists := ratings.Get(caller.String()); !exists {
		panic("Rating either unrecognisable/ inaccessible from your address")
	}
}

func findRatingByID(id seqid.ID) (string, bool) {
	keyP := ""
	found := false

	ratings.Iterate("", "", func(key string, value any) bool {
		c := value.(*comment.Comment)
		if c.ID() == id {
			keyP = key
			found = true
			return true
		}
		return false
	})

	return keyP, found
}

func RenderRatings() string {
	out := md.H2("★ Ratings ★\n\n")
	out += md.Link("[ ADD A RATING ]", txlink.Call("NewRatingComment", "comment", "Comment", "rating", "0"))
	out += "\n\n"

	if ratings.Size() == 0 {
		out += "No ratings found."
	} else {
		ratings.ReverseIterate("", "", func(key string, value interface{}) bool {
			comment := value.(*comment.Comment)
			user := users.ResolveAddress(comment.Author())
			if user != nil {
				out += md.Bold(ufmt.Sprintf("[#%s]:", user.RenderLink("")))
			} else {
				out += md.Bold(ufmt.Sprintf("[#%s]:", key))
			}

			out += ufmt.Sprintf(" (%d) ", comment.Rating())
			for i := comment.Rating(); i > 0; i-- {
				out += "★"
			}
			if comment.Comment() != nil {
				out += ufmt.Sprintf(" %s [ ", *comment.Comment())
			}
			out += md.Link("✎ EDIT", txlink.Call("UpdateByID", "comment", "Comment", "id", comment.ID().String(), "rating", "0"))
			out += " | "
			out += md.Link("🗑 DELETE", txlink.Call("DeleteRating", "id", comment.ID().String()))
			out += " ]\n\n"
			return false
		})
	}

	return out
}
