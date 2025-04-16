package card

import (
	"strings"

	"gno.land/p/moul/md"

	"gno.land/p/lou/ascii"
)

type Card struct {
	header          string
	hWidthAlignment ascii.WAlignment

	content          []string
	cWidthAlignment  ascii.WAlignment
	cHeightAlignment ascii.HAlignment

	width  uint64
	height uint64

	footer *string
	link   *string
}

func (card *Card) SetHeader(header string) {
	card.header = header
}

func (card *Card) SetContent(content []string) {
	card.content = content
}

func (card *Card) AppendContent(line string) {
	card.content = append(card.content, line)
}

func (card *Card) SetHeight(height uint64) {
	card.height = height
}

func (card *Card) SetWidth(width uint64) {
	card.width = width
}

func (card *Card) SetHeaderAlignmentW(
	width ascii.WAlignment,
) {
	card.hWidthAlignment = width
}

func (card *Card) SetContentAlignmentW(
	width ascii.WAlignment,
) {
	card.cWidthAlignment = width
}

func (card *Card) SetContentAlignmentH(
	height ascii.HAlignment,
) {
	card.cHeightAlignment = height
}

func contentRender(lines []string, wAlign ascii.WAlignment, width int, height int, hAlign ascii.HAlignment) string { // TODO make fixed frame
	var wrapped []string

	for _, line := range lines {
		words := strings.Fields(line)
		current := ""
		for _, word := range words {
			if len(current)+len(word)+1 > width {
				wrapped = append(wrapped, current)
				current = word
			} else {
				if current == "" {
					current = word
				} else {
					current += " " + word
				}
			}
		}
		if current != "" {
			wrapped = append(wrapped, current)
		}
	}

	wrapped = ascii.PadHeight(wrapped, height, hAlign)

	body := ""
	for _, line := range wrapped {
		body += ascii.PadLine(line, width, wAlign, "⠀") + "\n"
	}

	return body + "\n"
}

func (card Card) String() string {
	header := ascii.PadLine(card.header, int(card.width)/2, card.hWidthAlignment, "⠀")
	out := md.H1(header) + "\n\n"
	formattedContent := contentRender(card.content, card.cWidthAlignment, int(card.width), int(card.height), card.cHeightAlignment)
	out += formattedContent + "\n\n"
	return out
}

func NewCard(
	header string,
	content []string,
	width,
	height uint64,
) *Card {
	maxW := uint64(len(header))
	if maxW > width {
		width = maxW
	}

	return &Card{
		header:          header,
		hWidthAlignment: ascii.AlignLeft,

		content:          content,
		cWidthAlignment:  ascii.AlignLeft,
		cHeightAlignment: ascii.AlignTop,

		width:  width,
		height: height,
	}
}
