package main

import "strconv"

type Color int

const (
	Black Color = iota + 1
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

func Text(c Color) string {
	return "\x1b[" + strconv.Itoa(int(c)+29) + "m"
}
func BrightText(c Color) string {
	return "\x1b[1;" + strconv.Itoa(int(c)+29) + "m"
}
func All(foreground Color, bright bool, background Color) string {
	if bright {
		return "\x1b[1;" + strconv.Itoa(int(foreground)+29) + ";" + strconv.Itoa(int(background)+39) + "m"
	} else {
		return "\x1b[" + strconv.Itoa(int(foreground)+29) + ";" + strconv.Itoa(int(background)+39) + "m"
	}
}
func Reset() string {
	return "\x1b[m"
}
