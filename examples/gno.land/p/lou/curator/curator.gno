package curator

import (
	"errors"
	"std"
	"strings"

	"gno.land/p/demo/seqid"
	"gno.land/p/demo/ufmt"
	"gno.land/p/lou/card"
	"gno.land/p/moul/md"
)

var (
	ErrCollectionExists   = errors.New("Collection already exists")
	ErrNotCollectionOwner = errors.New("Only the owner can modify this collection")
	ErrCollectionMissing  = errors.New("Collection not found")
	ErrItemAlrExists      = errors.New("An item with this title already exists in the collection")
	ErrItemAlrLinked      = errors.New("An item with this path already exists in the collection")
)

type Collection struct {
	id          seqid.ID
	title       string
	description string
	owner       std.Address
	items       []*card.Card
	tags        []string
}

func (c Collection) ID() seqid.ID {
	return c.id
}

func (c Collection) Title() string {
	return c.title
}

func (c Collection) Description() string {
	return c.description
}

func (c Collection) Owner() std.Address {
	return c.owner
}

func (c Collection) Items() []*card.Card {
	return c.items
}

func (c Collection) Tags() []string {
	return c.tags
}

func (c Collection) PreviewCollection() string {
	path := ufmt.Sprintf("/r/lou/collections:%s", c.ID().String())
	formattedPath := md.Link("➔", path)
	tags := formatTags(c.tags)

	card := card.NewCard(c.title, []string{c.description, tags, "\n", formattedPath}, 24, 0)
	return card.String()
}

func AddToCollection(c *Collection, title, path string, tags []string) (*Collection, error) {
	if c.owner != std.PreviousRealm().Address() {
		return nil, ErrNotCollectionOwner
	}
	formattedPath := strings.TrimPrefix(path, "gno.land")

	for _, item := range c.Items() {
		if item.Header() == title {
			return nil, ErrItemAlrExists
		}

		for _, line := range item.Content() {
			if strings.Contains(line, formattedPath) {
				return nil, ErrItemAlrLinked
			}
		}
	}
	item := createCard(title, path, tags)
	c.items = append(c.items, item)

	return c, nil
}

func RenderCollection(c *Collection) string {
	out := md.H1(c.title) + "\n\n"
	out += md.HorizontalRule() + "\n\n"

	out += "<gno-columns>\n\n"
	for i, c := range c.items {
		out += c.String()
		out += md.HorizontalRule()
		if i%3 == 0 {
		} else {
			out += "|||\n\n"
		}
	}
	out += "</gno-columns>\n\n"

	return out
}

func NewCollection(title,
	description string,
	tags []string,
	id seqid.ID,
	owner std.Address,
) (*Collection, error) {
	return &Collection{
		id:          id,
		title:       title,
		description: description,
		owner:       owner,
		items:       []*card.Card{},
		tags:        tags,
	}, nil
}

func CreateCollection(
	id seqid.ID,
	title,
	description string,
	tags []string,
	owner std.Address,
) (*Collection, error) {
	col, err := NewCollection(title, description, tags, id, owner)
	if err != nil {
		return col, err
	}
	return col, nil
}

func formatTags(tags []string) string {
	out := "- Tags : "
	for i, tag := range tags {
		out += md.InlineCode(tag)
		if i != len(tags)-1 {
			out += ", "
		}
	}
	return out
}

func createCard(title, path string, tags []string) *card.Card {
	namespace := strings.Split(path, "/")[2]
	formattedPath := strings.TrimPrefix(path, "gno.land")
	link := md.Link("➔", formattedPath)
	formattedTags := formatTags(tags)

	return card.NewCard(title, []string{"by " + namespace, formattedPath, link, formattedTags}, 24, 0)
}
