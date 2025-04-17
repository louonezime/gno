package collections

import (
	"std"
	"strings"

	"gno.land/p/demo/avl"
	"gno.land/p/demo/seqid"
	"gno.land/p/moul/md"
	"gno.land/p/moul/txlink"
	"gno.land/r/sys/users"

	"gno.land/p/lou/curator"
)

var (
	collections  *avl.Tree
	collectionID seqid.ID
)

func init() {
	collections = avl.NewTree()
	collectionID = seqid.ID(0)
}

func NewCollection(title, description string, tags string) string {
	caller := std.PreviousRealm().Address()
	splitTags := strings.Fields(tags)
	id := collectionID.Next()

	c, err := curator.CreateCollection(id, title, description, splitTags, caller)
	if err != nil {
		panic(err)
	}

	collections.Set(id.String(), c)
	return "Successfully created collection! " + id.String()
}

func UpdateCollection(id, title, description string, tags []string) string {
	caller := std.PreviousRealm().Address()

	val, removed := collections.Remove(id)
	old := val.(*curator.Collection)
	if !removed {
		panic("Update failed; couldn't remove old collection")
	}

	newCollection, err := curator.CreateCollection(old.ID(), title, description, tags, caller)
	if err != nil {
		panic(err)
	}

	collections.Set(caller.String(), newCollection)
	return "Successfully updated collection."
}

func DeleteCollection() string {
	caller := std.PreviousRealm().Address()

	if _, removed := collections.Remove(caller.String()); !removed {
		panic("Couldn't delete collection.")
	}
	return "Collection deleted."
}

func AddToCollection(id, title, path string, tags string) string {
	splitTags := strings.Fields(tags)

	val, exists := collections.Get(id)
	if !exists {
		panic("Collection is unrecognisable.")
	}
	collection := val.(*curator.Collection)

	c, err := curator.AddToCollection(collection, title, path, splitTags)
	if err != nil {
		panic(err)
	}
	if _, removed := collections.Remove(c.ID().String()); !removed {
		panic("Adding of collection failed; couldn't remove current collection")
	}
	collections.Set(c.ID().String(), c)
	return title + " successfully added to collection."
}

func renderCollection(id string) string {
	if collection, exists := collections.Get(id); exists {
		c := collection.(*curator.Collection)
		out := curator.RenderCollection(c)

		out += md.Link("[ ADD TO COLLECTION ]", txlink.Call("AddToCollection", "id", c.ID().String()))
		return out
	}
	return "Collection is unrecognisable."
}

func Render(path string) string {
	if path != "" {
		return renderCollection(path)
	}
	out := md.H1("Collections") + "\n\n"
	out += md.HorizontalRule()
	// games, err := curator.CreateCollection(id, "Games", "jjj", []string{"fun", "entertainment"}, std.PreviousRealm().Address())
	// if err != nil {
	// 	panic(err)
	// }
	// games.AddToCollection("Tamagotchi", "gno.land/p/demo/tamagotchi", []string{"game", "fun", "pet"})
	// collections.Set(games.ID().String(), games)

	if collections.Size() == 0 {
		out += "No collections found."
	} else {
		collections.ReverseIterate("", "", func(key string, value interface{}) bool {
			c := value.(*curator.Collection)
			user := users.ResolveAddress(c.Owner())

			out += c.PreviewCollection()
			out += "\nfrom "
			if user != nil {
				out += md.Bold(user.RenderLink(""))
			} else {
				out += md.Bold(c.Owner().String())
			}

			out += " [" + md.Link("✎ EDIT", txlink.Call("UpdateCollection", "title", c.Title(), "desc", c.Description()))
			out += " | " + md.Link("🗑 DELETE", txlink.Call("DeleteCollection"))
			out += " ]\n\n"
			out += md.HorizontalRule()
			return false
		})
	}

	return out
}
