package ascii

import (
	"strconv"
	"strings"

	"gno.land/p/moul/md"
)

const (
	// Width Alignment
	Center = "center"
	Left   = "left"
	Right  = "right"

	// Height Alignment
	Bottom = "bottom"
	Top    = "top"
)

// padLine aligns text within a given width.
//
// Supports Left, Right, and Center alignment.
func padLine(line string, width int, align string) string {
	padding := width - len(line)
	switch align {
	case Right:
		return Repeat(" ", padding) + line
	case Center:
		left := padding / 2
		right := padding - left
		return Repeat(" ", left) + line + Repeat(" ", right)
	default: // Left
		return line + Repeat(" ", padding)
	}
}

// padHeight pads lines vertically according to alignment.
//
// Supports Top, Bottom, and Center alignment.
func padHeight(lines []string, height int, align string) []string {
	padded := []string{}
	extra := height - len(lines)

	topPad := 0
	switch align {
	case Bottom:
		topPad = extra
	case Center:
		topPad = extra / 2
	}

	for i := 0; i < topPad; i++ {
		padded = append(padded, "")
	}
	padded = append(padded, lines...)
	for len(padded) < height {
		padded = append(padded, "")
	}
	return padded
}

// Repeat returns repetition of a string n times.
func Repeat(char string, n int) string {
	out := ""

	for i := 0; i < n; i++ {
		out += char
	}
	return out
}

// Box draws a single-line text in a simple box.
// If the string contains newlines, it falls back to FlexFrame function.
//
// Example:
//
//	Box("Hello World\n!")
//
// Gives:
//
//	+-------------+
//	| Hello World |
//	| !           |
//	+-------------+
func Box(text string) string {
	width := len(text) + 4

	if strings.Contains(text, "\n") {
		return FlexFrame(strings.Split(text, "\n"), "left")
	}
	top := "+" + Repeat("-", width-2) + "+\n"
	middle := "| " + text + " |\n"
	bottom := "+" + Repeat("-", width-2) + "+"

	return md.CodeBlock(top+middle+bottom) + "\n"
}

// FlexFrame draws a frame with automatic width and alignment.
//
// Example:
//
//	FlexFrame([]string{"hello", "worldd", "!!"}, "right")
//
// Gives:
//
//	+-------+
//	|  hello |
//	| worldd |
//	|     !! |
//	+-------+
func FlexFrame(lines []string, align string) string {
	maxWidth := 0

	for i := 0; i < len(lines); i++ {
		if len(lines[i]) > maxWidth {
			maxWidth = len(lines[i])
		}
	}

	top := "+" + Repeat("-", maxWidth+2) + "+\n"
	bottom := "+" + Repeat("-", maxWidth+2) + "+"

	body := ""
	for i := 0; i < len(lines); i++ {
		body += "| " + padLine(lines[i], maxWidth, align) + " |\n"
	}

	return md.CodeBlock(top+body+bottom) + "\n"
}

// Frame draws a frame with specific width, height and alignment options.
//
// Example:
//
//	Frame([]string{"hello", "world", "!!"}, "center", 10, 5, "center")
//
// Gives:
//
//	+------------+
//	|            |
//	|   hello    |
//	|   world    |
//	|     !!     |
//	|            |
//	+------------+
func Frame(lines []string, wAlign string, width int, height int, hAlign string) string {
	maxWidth := width
	for i := 0; i < len(lines); i++ {
		if len(lines[i]) > maxWidth {
			maxWidth = len(lines[i])
		}
	}

	if len(lines) > height {
		height = len(lines)
	}
	lines = padHeight(lines, height, hAlign)

	top := "+" + Repeat("-", maxWidth+2) + "+\n"
	bottom := "+" + Repeat("-", maxWidth+2) + "+"
	body := ""

	for _, line := range lines {
		body += "| " + padLine(line, maxWidth, wAlign) + " |\n"
	}

	return md.CodeBlock(top+body+bottom) + "\n"
}

// PercentageBar renders a visual progress bar.
//
// Example:
//
//	ProgressBar(2, 6, 10, true)
//
// Gives: [###-------] 33%
func ProgressBar(current int, total int, size int, show bool) string {
	if total == 0 {
		return PercentageBar(0, size, show)
	}
	percent := (current * 100) / total

	return PercentageBar(percent, size, show)
}

// PercentageBar renders a visual progress bar.
//
// Example:
//
//	PercentageBar(50, 6, false)
//
// Gives: [###---]
func PercentageBar(percent int, size int, show bool) string {
	fillLength := (percent * size) / 100
	emptyLength := size - fillLength

	filled := Repeat("#", fillLength)
	empty := Repeat("-", emptyLength)

	out := "[" + filled + empty + "]"
	if !show {
		return out
	}
	return out + " " + strconv.Itoa(percent) + "%"
}

// Grid renders a 2D grid of characters.
//
// Example:
//
//	Grid(3, 3, "x")
//
// Gives:
//
//	xxx
//	xxx
//	xxx
func Grid(rows int, cols int, fill string) string {
	out := ""

	for r := 0; r < rows; r++ {
		row := ""
		for c := 0; c < cols; c++ {
			row += fill
		}
		out += row + "\n"
	}
	return out
}
