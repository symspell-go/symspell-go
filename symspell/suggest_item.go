package symspell

// SuggestItem represents a spelling suggestion returned from Lookup.
type SuggestItem struct {
	term     string
	distance int
	count    int64
}

// SuggestItems is a slice of SuggestItem, used for sorting.
type SuggestItems []SuggestItem

func (s SuggestItems) Len() int {
	return len(s)
}

func (s SuggestItems) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s SuggestItems) Less(i, j int) bool {
	if s[i].distance == s[j].distance {
		return s[i].count > s[j].count
	}
	return s[i].distance < s[j].distance
}
