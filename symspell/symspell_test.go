package symspell

import (
	"testing"
)

func Test_WordsWithSharedPrefixShouldRetainCounts(t *testing.T) {
	var symSpell, _ = NewSymSpell(16, 1, 3, 1, 5)

	symSpell.CreateDictionaryEntry("pipe", 5, nil)
	symSpell.CreateDictionaryEntry("pips", 10, nil)

	{
		result := symSpell.Lookup("pip", All, 1, false)
		equal(t, 2, len(result))
		equal(t, "pips", result[0].term)
		equal(t, 10, result[0].count)
		equal(t, "pipe", result[1].term)
		equal(t, 5, result[1].count)
	}

	{
		result := symSpell.Lookup("pipe", All, 1, false)
		equal(t, len(result), 2)
		equal(t, result[0].term, "pipe")
		equal(t, result[0].count, 5)
		equal(t, result[0].distance, 0)
		equal(t, result[1].term, "pips")
		equal(t, result[1].count, 10)
	}

	{
		result := symSpell.Lookup("pips", All, 1, false)
		equal(t, 2, len(result))
		equal(t, "pips", result[0].term)
		equal(t, 10, result[0].count)
		equal(t, "pipe", result[1].term)
		equal(t, 5, result[1].count)
	}
}

func Test_VerbosityShouldControlLookupResults(t *testing.T) {
	symSpell, _ := NewSymSpell(16, 2, 3, 1, 5)

	symSpell.CreateDictionaryEntry("steam", 1, nil)
	symSpell.CreateDictionaryEntry("steams", 2, nil)
	symSpell.CreateDictionaryEntry("steem", 3, nil)

	{
		result := symSpell.Lookup("steems", Top, 2, false)
		equal(t, 1, result.Len())
	}
	{
		result := symSpell.Lookup("steems", Closest, 2, false)
		equal(t, 2, result.Len())
	}
	{
		result := symSpell.Lookup("steems", All, 2, false)
		equal(t, 3, result.Len())
	}
}

func Test_LookupShouldReturnMostFrequent(t *testing.T) {
	symSpell, _ := NewSymSpell(16, 2, 3, 1, 5)

	symSpell.CreateDictionaryEntry("steama", 4, nil)
	symSpell.CreateDictionaryEntry("steamb", 6, nil)
	symSpell.CreateDictionaryEntry("steamc", 2, nil)

	result := symSpell.Lookup("steam", Top, 2, false)
	equal(t, 1, result.Len())
	equal(t, "steamb", result[0].term)
	equal(t, 6, result[0].count)
}

func Test_LookupShouldFindExactMatch(t *testing.T) {
	symSpell, _ := NewSymSpell(16, 2, 3, 1, 5)

	symSpell.CreateDictionaryEntry("steama", 4, nil)
	symSpell.CreateDictionaryEntry("steamb", 6, nil)
	symSpell.CreateDictionaryEntry("steamc", 2, nil)

	result := symSpell.Lookup("steama", Top, 2, false)
	equal(t, 1, result.Len())
	equal(t, "steama", result[0].term)
}

func Test_LookupShouldNotReturnNonWordDelete(t *testing.T) {
	symSpell, _ := NewSymSpell(16, 2, 7, 1, 5)

	symSpell.CreateDictionaryEntry("pawn", 10, nil)

	{
		result := symSpell.Lookup("paw", Top, 0, false)
		equal(t, 0, result.Len())
	}

	{
		result := symSpell.Lookup("awn", Top, 0, false)
		equal(t, 0, result.Len())
	}
}

func Test_LookupShouldNotReturnLowCountWord(t *testing.T) {
	symSpell, _ := NewSymSpell(16, 2, 7, 10, 5)

	symSpell.CreateDictionaryEntry("pawn", 1, nil)

	{
		result := symSpell.Lookup("pawn", Top, 0, false)
		equal(t, 0, result.Len())
	}
}

func Test_LookupShouldNotReturnLowCountWordThatsAlsoDeleteWord(t *testing.T) {
	symSpell, _ := NewSymSpell(16, 2, 7, 10, 5)

	symSpell.CreateDictionaryEntry("flame", 20, nil)
	symSpell.CreateDictionaryEntry("flam", 1, nil)

	{
		result := symSpell.Lookup("flam", Top, 0, false)
		equal(t, 0, result.Len())
	}
}

func equal[T comparable](t *testing.T, a, b T) {
	t.Helper()
	if a == b {
		return
	}
	t.Errorf("Expected %v, got %v", a, b)
}
