package symspell

import (
	"bufio"
	"errors"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Verbosity controls the quantity/closeness of the returned spelling suggestions.
type Verbosity int

const (
	// Top suggestion with the highest term frequency of the suggestions of smallest edit distance found.
	Top Verbosity = iota
	// All suggestions of smallest edit distance found, suggestions ordered by term frequency.
	Closest
	// All suggestions within maxEditDistance, suggestions ordered by edit distance
	// , then by term frequency (slower, no early termination).
	All
)

const (
	defaultMaxEditDistance = 2
	defaultPrefixLength    = 7
	defaultCountThreshold  = 1
	defaultInitialCapacity = 16
	defaultCompactLevel    = 5
)

// SymSpell is the main struct for the SymSpell spelling correction algorithm.
type SymSpell struct {
	initialCapacity           int
	maxDictionaryEditDistance int
	prefixLength              int
	countThreshold            int64
	compactMask               uint32
	maxDictionaryWordLength   int

	deletes             map[int]map[string]struct{}
	words               map[string]int64
	belowThresholdWords map[string]int64

	bigrams        map[string]int64
	bigramCountMin int64
	n              float64
}

// NewSymSpell creates a new instance of SymSpell.
func NewSymSpell(initialCapacity int, maxDictionaryEditDistance int, prefixLength int, countThreshold int64, compactLevel uint8) (*SymSpell, error) {
	if initialCapacity < 0 {
		return nil, errors.New("initialCapacity must be >= 0")
	}
	if maxDictionaryEditDistance < 0 {
		return nil, errors.New("maxDictionaryEditDistance must be >= 0")
	}
	if prefixLength < 1 || prefixLength <= maxDictionaryEditDistance {
		return nil, errors.New("prefixLength must be > 1 and > maxDictionaryEditDistance")
	}
	if countThreshold < 0 {
		return nil, errors.New("countThreshold must be >= 0")
	}
	if compactLevel > 16 {
		return nil, errors.New("compactLevel must be <= 16")
	}
	compactMask := uint32(math.MaxUint32>>(3+compactLevel)) << 2
	symSpell := &SymSpell{
		initialCapacity:           initialCapacity,
		maxDictionaryEditDistance: maxDictionaryEditDistance,
		prefixLength:              prefixLength,
		countThreshold:            countThreshold,
		compactMask:               compactMask,
		deletes:                   make(map[int]map[string]struct{}),
		words:                     make(map[string]int64, initialCapacity),
		belowThresholdWords:       make(map[string]int64),
		bigrams:                   make(map[string]int64),
		bigramCountMin:            math.MaxInt64,
		n:                         1024908267229.0,
	}
	return symSpell, nil
}

// CreateDictionaryEntry creates or updates an entry in the dictionary.
func (s *SymSpell) CreateDictionaryEntry(key string, count int64, staging *SuggestionStage) bool {
	if count <= 0 {
		if s.countThreshold > 0 {
			return false
		}
		count = 0
	}
	var countPrevious int64

	if s.countThreshold > 1 {
		if c, found := s.belowThresholdWords[key]; found {
			countPrevious = c
			if math.MaxInt64-countPrevious > count {
				count += countPrevious
			} else {
				count = math.MaxInt64
			}
			if count >= s.countThreshold {
				delete(s.belowThresholdWords, key)
			} else {
				s.belowThresholdWords[key] = count
				return false
			}
		} else if c, found := s.words[key]; found {
			countPrevious = c
			if math.MaxInt64-countPrevious > count {
				count += countPrevious
			} else {
				count = math.MaxInt64
			}
			s.words[key] = count
			return false
		} else if count < s.countThreshold {
			s.belowThresholdWords[key] = count
			return false
		}
	} else {
		if c, found := s.words[key]; found {
			countPrevious = c
			if math.MaxInt64-countPrevious > count {
				count += countPrevious
			} else {
				count = math.MaxInt64
			}
			s.words[key] = count
			return false
		} else if count < s.countThreshold {
			s.belowThresholdWords[key] = count
			return false
		}
	}

	s.words[key] = count

	if len(key) > s.maxDictionaryWordLength {
		s.maxDictionaryWordLength = len(key)
	}

	edits := s.EditsPrefix(key)

	if staging != nil {
		for deleteStr := range edits {
			staging.Add(s.GetStringHash(deleteStr), key)
		}
	} else {
		for deleteStr := range edits {
			deleteHash := s.GetStringHash(deleteStr)
			if s.deletes[deleteHash] == nil {
				s.deletes[deleteHash] = make(map[string]struct{})
			}
			s.deletes[deleteHash][key] = struct{}{}
		}
	}
	return true
}

// EditsPrefix generates all possible deletes for a word up to maxEditDistance.
func (s *SymSpell) EditsPrefix(key string) map[string]struct{} {
	hashSet := make(map[string]struct{})
	if len(key) <= s.maxDictionaryEditDistance {
		hashSet[""] = struct{}{}
	}
	if len(key) > s.prefixLength {
		key = key[:s.prefixLength]
	}
	hashSet[key] = struct{}{}
	s.Edits(key, 0, hashSet)
	return hashSet
}

// Edits recursively generates all possible deletes for a word.
func (s *SymSpell) Edits(word string, editDistance int, deleteWords map[string]struct{}) {
	editDistance++
	if len(word) > 1 {
		for i := 0; i < len(word); i++ {
			deleteStr := word[:i] + word[i+1:]
			if _, exists := deleteWords[deleteStr]; !exists {
				deleteWords[deleteStr] = struct{}{}
				if editDistance < s.maxDictionaryEditDistance {
					s.Edits(deleteStr, editDistance, deleteWords)
				}
			}
		}
	}
}

// GetStringHash computes a hash value for a string.
func (s *SymSpell) GetStringHash(str string) int {
	lenRunes := 0
	for range str {
		lenRunes++
	}
	lenMask := lenRunes
	if lenMask > 3 {
		lenMask = 3
	}

	var hash uint32 = 2166136261
	for _, r := range str {
		hash ^= uint32(r)
		hash *= 16777619
	}

	hash &= s.compactMask
	hash |= uint32(lenMask)
	return int(hash)
}

// CommitStaged commits staged dictionary additions.
func (s *SymSpell) CommitStaged(staging *SuggestionStage) {
	staging.CommitTo(s.deletes)
}

// LoadDictionary loads multiple dictionary entries from a file of word/frequency count pairs.
func (s *SymSpell) LoadDictionary(corpus string, termIndex int, countIndex int, separatorChars string) (bool, error) {
	file, err := os.Open(corpus)
	if err != nil {
		return false, err
	}
	defer file.Close()
	return s.LoadDictionaryFromReader(file, termIndex, countIndex, separatorChars)
}

// LoadDictionaryFromReader loads dictionary entries from an io.Reader.
func (s *SymSpell) LoadDictionaryFromReader(reader io.Reader, termIndex int, countIndex int, separatorChars string) (bool, error) {
	staging := NewSuggestionStage(16384)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		var lineParts []string
		if separatorChars == "" {
			lineParts = strings.Fields(line)
		} else {
			lineParts = strings.Split(line, separatorChars)
		}
		if len(lineParts) >= 2 {
			key := lineParts[termIndex]
			count, err := strconv.ParseInt(lineParts[countIndex], 10, 64)
			if err == nil {
				s.CreateDictionaryEntry(key, count, staging)
			}
		}
	}
	if s.deletes == nil {
		s.deletes = make(map[int]map[string]struct{}, staging.DeleteCount())
	}
	s.CommitStaged(staging)
	return true, nil
}

func (s *SymSpell) Lookup(input string, verbosity Verbosity, maxEditDistance int, includeUnknown bool) SuggestItems {
	// verbosity=Top: the suggestion with the highest term frequency of the suggestions of smallest edit distance found
	// verbosity=Closest: all suggestions of smallest edit distance found, the suggestions are ordered by term frequency
	// verbosity=All: all suggestions <= maxEditDistance, the suggestions are ordered by edit distance, then by term frequency (slower, no early termination)

	// maxEditDistance used in Lookup can't be bigger than the maxDictionaryEditDistance
	// used to construct the underlying dictionary structure.
	if maxEditDistance > s.maxDictionaryEditDistance {
		panic("maxEditDistance > maxDictionaryEditDistance")
	}

	suggestions := SuggestItems{}
	inputLen := len(input)
	// quick look for exact match
	var suggestionCount int64
	var ok bool

	// deletes we've considered already
	hashset1 := make(map[string]struct{})
	// suggestions we've considered already
	hashset2 := make(map[string]struct{})

	maxEditDistance2 := maxEditDistance
	candidatePointer := 0
	candidates := []string{}

	// add original prefix
	inputPrefixLen := inputLen

	distanceComparer := NewDistanceComparer()

	// early exit - word is too big to possibly match any words
	if inputLen-maxEditDistance > s.maxDictionaryWordLength {
		goto end
	}

	if suggestionCount, ok = s.words[input]; ok {
		suggestions = append(suggestions, SuggestItem{term: input, distance: 0, count: suggestionCount})
		// early exit - return exact match, unless caller wants all matches
		if verbosity != All {
			goto end
		}
	}

	// early termination, if we only want to check if word in dictionary or get its frequency e.g. for word segmentation
	if maxEditDistance == 0 {
		goto end
	}

	// we considered the input already in the word.TryGetValue above
	hashset2[input] = struct{}{}

	if inputPrefixLen > s.prefixLength {
		inputPrefixLen = s.prefixLength
		candidates = append(candidates, input[:inputPrefixLen])
	} else {
		candidates = append(candidates, input)
	}

	for candidatePointer < len(candidates) {
		candidate := candidates[candidatePointer]
		candidatePointer++
		candidateLen := len(candidate)
		lengthDiff := inputPrefixLen - candidateLen

		// save some time - early termination
		// if candidate distance is already higher than suggestion distance, then there are no better suggestions to be expected
		if lengthDiff > maxEditDistance2 {
			// skip to next candidate if VerbosityAll, look no further if VerbosityTop or Closest
			// (candidates are ordered by delete distance, so none are closer than current)
			if verbosity == All {
				continue
			} else {
				break
			}
		}

		// read candidate entry from dictionary
		if dictSuggestions, found := s.deletes[s.GetStringHash(candidate)]; found {
			// iterate through suggestions (to other correct dictionary items) of delete item and add them to suggestion list
			for suggestion := range dictSuggestions {
				suggestionLen := len(suggestion)
				if suggestion == input {
					continue
				}
				if abs(suggestionLen-inputLen) > maxEditDistance2 ||
					suggestionLen < candidateLen ||
					(suggestionLen == candidateLen && suggestion != candidate) {
					continue
				}
				suggPrefixLen := min(suggestionLen, s.prefixLength)
				if suggPrefixLen > inputPrefixLen && (suggPrefixLen-candidateLen) > maxEditDistance2 {
					continue
				}

				distance := 0
				minLen := 0
				if candidateLen == 0 {
					// suggestions which have no common chars with input (inputLen<=maxEditDistance && suggestionLen<=maxEditDistance)
					distance = max(inputLen, suggestionLen)
					if distance > maxEditDistance2 || !addToSet(hashset2, suggestion) {
						continue
					}
				} else if suggestionLen == 1 {
					if !strings.ContainsRune(input, rune(suggestion[0])) {
						distance = inputLen
					} else {
						distance = inputLen - 1
					}
					if distance > maxEditDistance2 || !addToSet(hashset2, suggestion) {
						continue
					}
				} else if (s.prefixLength - maxEditDistance) == candidateLen {
					minLen = min(inputLen, suggestionLen) - s.prefixLength
					if (minLen > 1 && input[inputLen-minLen:] != suggestion[suggestionLen-minLen:]) ||
						(minLen > 0 &&
							input[inputLen-minLen] != suggestion[suggestionLen-minLen] &&
							(input[inputLen-minLen-1] != suggestion[suggestionLen-minLen] ||
								input[inputLen-minLen] != suggestion[suggestionLen-minLen-1])) {
						continue
					}
				} else {
					if (verbosity != All && !s.deleteInSuggestionPrefix(candidate, candidateLen, suggestion, suggestionLen)) ||
						!addToSet(hashset2, suggestion) {
						continue
					}
					distance = distanceComparer.Compare(input, suggestion, maxEditDistance2)
					if distance < 0 {
						continue
					}
				}

				// save some time
				// do not process higher distances than those already found, if verbosity<All (note: maxEditDistance2 will always equal maxEditDistance when VerbosityAll)
				if distance <= maxEditDistance2 {
					suggestionCount = s.words[suggestion]
					si := SuggestItem{term: suggestion, distance: distance, count: suggestionCount}
					if len(suggestions) > 0 {
						switch verbosity {
						case Closest:
							// we will calculate DamLev distance only to the smallest found distance so far
							if distance < maxEditDistance2 {
								suggestions = suggestions[:0]
							}
							break
						case Top:
							if distance < maxEditDistance2 || suggestionCount > suggestions[0].count {
								maxEditDistance2 = distance
								suggestions[0] = si
							}
							continue
						}
					}
					if verbosity != All {
						maxEditDistance2 = distance
					}
					suggestions = append(suggestions, si)
				}
			}
		}

		// add edits
		// derive edits (deletes) from candidate (input) and add them to candidates list
		// this is a recursive process until the maximum edit distance has been reached
		if lengthDiff < maxEditDistance && candidateLen <= s.prefixLength {
			// save some time
			// do not create edits with edit distance bigger than suggestions already found
			if verbosity != All && lengthDiff >= maxEditDistance2 {
				continue
			}

			for i := 0; i < candidateLen; i++ {
				delete := candidate[:i] + candidate[i+1:]

				if _, found := hashset1[delete]; !found {
					hashset1[delete] = struct{}{}
					candidates = append(candidates, delete)
				}
			}
		}
	}

	// sort by ascending edit distance, then by descending word frequency
	if len(suggestions) > 1 {
		sort.Sort(suggestions)

		uniqueSuggestions := make(SuggestItems, 0, len(suggestions))
		seen := make(map[string]struct{}, len(suggestions))
		for _, suggestion := range suggestions {
			if _, found := seen[suggestion.term]; found {
				continue
			}

			uniqueSuggestions = append(uniqueSuggestions, suggestion)
			seen[suggestion.term] = struct{}{}
		}

		suggestions = uniqueSuggestions
	}
end:
	if includeUnknown && len(suggestions) == 0 {
		suggestions = append(suggestions, SuggestItem{term: input, distance: maxEditDistance + 1, count: 0})
	}
	return suggestions
}

func (s *SymSpell) LookupCompound(input string, editDistanceMax int) SuggestItems {
	// Parse input string into single terms
	termList1 := parseWords(input)

	// Suggestions for a single term
	var suggestions SuggestItems
	// Suggestion parts
	suggestionParts := make(SuggestItems, 0)
	// Distance comparer
	distanceComparer := NewDistanceComparer()

	// Translate every term to its best suggestion, otherwise it remains unchanged
	lastCombi := false
	for i := 0; i < len(termList1); i++ {
		suggestions = s.Lookup(termList1[i], Top, editDistanceMax, false)

		// Combi check, always before split
		if i > 0 && !lastCombi {
			combinedTerm := termList1[i-1] + termList1[i]
			suggestionsCombi := s.Lookup(combinedTerm, Top, editDistanceMax, false)

			if len(suggestionsCombi) > 0 {
				best1 := suggestionParts[len(suggestionParts)-1]
				var best2 SuggestItem
				if len(suggestions) > 0 {
					best2 = suggestions[0]
				} else {
					// Unknown word
					best2.term = termList1[i]
					// Estimated edit distance
					best2.distance = editDistanceMax + 1
					// Estimated word occurrence probability P=10 / (n * 10^word length l)
					best2.count = int64(10 / math.Pow(10, float64(len(best2.term))))
				}

				// Distance1 = edit distance between 2 split terms and their best corrections
				distance1 := best1.distance + best2.distance
				if distance1 >= 0 && ((suggestionsCombi[0].distance+1 < distance1) || ((suggestionsCombi[0].distance+1 == distance1) && float64(suggestionsCombi[0].count) > float64(best1.count)/s.n*float64(best2.count))) {
					suggestionsCombi[0].distance++
					suggestionParts[len(suggestionParts)-1] = suggestionsCombi[0]
					lastCombi = true
					continue
				}
			}
		}
		lastCombi = false

		// Always split terms without suggestion / never split terms with suggestion ed=0 / never split single char terms
		if len(suggestions) > 0 && (suggestions[0].distance == 0 || len(termList1[i]) == 1) {
			// Choose best suggestion
			suggestionParts = append(suggestionParts, suggestions[0])
		} else {
			// If no perfect suggestion, split word into pairs
			var suggestionSplitBest *SuggestItem

			// Add original term
			if len(suggestions) > 0 {
				tmp := suggestions[0]
				suggestionSplitBest = &tmp
			}

			if len(termList1[i]) > 1 {
				for j := 1; j < len(termList1[i]); j++ {
					part1 := termList1[i][:j]
					part2 := termList1[i][j:]
					suggestionSplit := SuggestItem{}
					suggestions1 := s.Lookup(part1, Top, editDistanceMax, false)
					if len(suggestions1) > 0 {
						suggestions2 := s.Lookup(part2, Top, editDistanceMax, false)
						if len(suggestions2) > 0 {
							// Select best suggestion for split pair
							suggestionSplit.term = suggestions1[0].term + " " + suggestions2[0].term

							distance2 := distanceComparer.Compare(termList1[i], suggestionSplit.term, editDistanceMax)
							if distance2 < 0 {
								distance2 = editDistanceMax + 1
							}

							if suggestionSplitBest != nil {
								if distance2 > suggestionSplitBest.distance {
									continue
								}
								if distance2 < suggestionSplitBest.distance {
									suggestionSplitBest = nil
								}
							}

							suggestionSplit.distance = distance2
							// If bigram exists in bigram dictionary
							bigramCount, bigramExists := s.bigrams[suggestionSplit.term]
							if bigramExists {
								suggestionSplit.count = bigramCount

								// Increase count if split corrections are part of or identical to input
								// Single term correction exists
								if len(suggestions) > 0 {
									if suggestions1[0].term+suggestions2[0].term == termList1[i] {
										// Make count bigger than count of single term correction
										suggestionSplit.count = maxInt64(suggestionSplit.count, suggestions[0].count+2)
									} else if suggestions1[0].term == suggestions[0].term || suggestions2[0].term == suggestions[0].term {
										// Make count bigger than count of single term correction
										suggestionSplit.count = maxInt64(suggestionSplit.count, suggestions[0].count+1)
									}
								} else if suggestions1[0].term+suggestions2[0].term == termList1[i] {
									// No single term correction exists
									suggestionSplit.count = maxInt64(suggestionSplit.count, maxInt64(suggestions1[0].count, suggestions2[0].count)+2)
								}
							} else {
								// The Naive Bayes probability of the word combination is the product of the two word probabilities: P(AB) = P(A) * P(B)
								// Use it to estimate the frequency count of the combination
								suggestionSplit.count = minInt64(s.bigramCountMin, int64(float64(suggestions1[0].count)/s.n*float64(suggestions2[0].count)))
							}

							if suggestionSplitBest == nil || suggestionSplit.count > suggestionSplitBest.count {
								tmp := suggestionSplit
								suggestionSplitBest = &tmp
							}
						}
					}
				}

				if suggestionSplitBest != nil {
					// Select best suggestion for split pair
					suggestionParts = append(suggestionParts, *suggestionSplitBest)
				} else {
					si := SuggestItem{
						term:     termList1[i],
						count:    int64(10 / math.Pow(10, float64(len(termList1[i])))),
						distance: editDistanceMax + 1,
					}
					suggestionParts = append(suggestionParts, si)
				}
			} else {
				si := SuggestItem{
					term:     termList1[i],
					count:    int64(10 / math.Pow(10, float64(len(termList1[i])))),
					distance: editDistanceMax + 1,
				}
				suggestionParts = append(suggestionParts, si)
			}
		}
	}

	suggestion := SuggestItem{}

	count := s.n
	var sb strings.Builder
	for _, si := range suggestionParts {
		sb.WriteString(si.term)
		sb.WriteString(" ")
		count *= float64(si.count) / s.n
	}
	suggestion.count = int64(count)
	suggestion.term = strings.TrimSpace(sb.String())
	suggestion.distance = distanceComparer.Compare(input, suggestion.term, math.MaxInt32)

	suggestionsLine := SuggestItems{suggestion}
	return suggestionsLine
}

func (s *SymSpell) deleteInSuggestionPrefix(delete string, deleteLen int, suggestion string, suggestionLen int) bool {
	if deleteLen == 0 {
		return true
	}
	if s.prefixLength < suggestionLen {
		suggestionLen = s.prefixLength
	}
	j := 0
	for i := 0; i < deleteLen; i++ {
		delChar := delete[i]
		for j < suggestionLen && delChar != suggestion[j] {
			j++
		}
		if j == suggestionLen {
			return false
		}
	}
	return true
}
