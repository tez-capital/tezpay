package constants

import "slices"

var (
	LOG_TOP_LEVEL_HIDDEN_FIELDS = []string{
		"stage",
		"phase",
	}
)

func init() {
	slices.Sort(LOG_TOP_LEVEL_HIDDEN_FIELDS)
}
