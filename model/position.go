package model

import (
	"strings"
)

type Position string

const (
	POS_UNKNOWN Position = "UNK"
	POS_QB      Position = "QB"
	POS_RB      Position = "RB"
	POS_WR      Position = "WR"
	POS_TE      Position = "TE"
	POS_DEF     Position = "DEF"
	POS_K       Position = "K"
)

func ParsePosition(pos string) Position {
	pos = strings.ToLower(pos)
	switch pos {
	case "qb":
		return POS_QB
	case "rb":
		return POS_RB
	case "fb":
		return POS_RB
	case "wr":
		return POS_WR
	case "te":
		return POS_TE
	case "def":
		return POS_DEF
	case "k":
		return POS_K
	default:
		return POS_UNKNOWN
	}
}
