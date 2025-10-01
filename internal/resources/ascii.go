package resources

import "strings"

// ASCII art characters for visual separators and diagrams
const (
	BoxCornerTopLeft     = "+"
	BoxCornerTopRight    = "+"
	BoxCornerBottomLeft  = "+"
	BoxCornerBottomRight = "+"
	BoxHorizontal        = "-"
	BoxVertical          = "|"
	Separator            = "----------------------------------------"
)

// GetBoxLine returns a box line of specified width
func GetBoxLine(width int) string {
	return BoxCornerTopLeft + strings.Repeat(BoxHorizontal, width-2) + BoxCornerTopRight
}

// GetSeparatorLine returns a separator line
func GetSeparatorLine() string {
	return Separator
}
