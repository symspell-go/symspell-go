package symspell

// DamerauOSA provides optimized methods for computing Damerau-Levenshtein Optimal String
// Alignment (OSA) comparisons between two strings.
type DamerauOSA struct {
	baseChar1Costs     []int
	basePrevChar1Costs []int
}

// NewDamerauOSA creates a new instance of DamerauOSA.
func NewDamerauOSA() *DamerauOSA {
	return &DamerauOSA{
		baseChar1Costs:     []int{},
		basePrevChar1Costs: []int{},
	}
}

// Distance computes and returns the Damerau-Levenshtein optimal string
// alignment edit distance between two strings.
// Returns -1 if the distance is greater than the maxDistance, 0 if the strings
// are equivalent, otherwise a positive number whose magnitude increases as
// difference between the strings increases.
func (d *DamerauOSA) Distance(string1, string2 string, maxDistance int) int {
	if string1 == "" || string2 == "" {
		return nullDistanceResults(string1, string2, maxDistance)
	}
	if maxDistance <= 0 {
		if string1 == string2 {
			return 0
		}
		return -1
	}
	iMaxDistance := maxDistance

	// Convert strings to rune slices to properly handle Unicode characters
	runeStr1 := []rune(string1)
	runeStr2 := []rune(string2)

	// Ensure shorter string is in runeStr1
	if len(runeStr1) > len(runeStr2) {
		runeStr1, runeStr2 = runeStr2, runeStr1
	}
	if len(runeStr2)-len(runeStr1) > iMaxDistance {
		return -1
	}

	// Identify common prefix and/or suffix that can be ignored
	len1, len2, start := prefixSuffixPrep(runeStr1, runeStr2)
	if len1 == 0 {
		if len2 <= iMaxDistance {
			return len2
		}
		return -1
	}

	// Resize cost arrays if necessary
	if len2 > len(d.baseChar1Costs) {
		d.baseChar1Costs = make([]int, len2)
		d.basePrevChar1Costs = make([]int, len2)
	}
	if iMaxDistance < len2 {
		return distanceWithMax(runeStr1, runeStr2, len1, len2, start, iMaxDistance, d.baseChar1Costs, d.basePrevChar1Costs)
	}
	return dist(runeStr1, runeStr2, len1, len2, start, d.baseChar1Costs, d.basePrevChar1Costs)
}

// dist is the internal implementation of the core Damerau-Levenshtein, optimal string alignment algorithm.
func dist(runeStr1, runeStr2 []rune, len1, len2, start int, char1Costs, prevChar1Costs []int) int {
	for j := 0; j < len2; j++ {
		char1Costs[j] = j + 1
	}
	var char1, prevChar1 rune
	var currentCost int
	for i := 0; i < len1; i++ {
		prevChar1 = char1
		char1 = runeStr1[start+i]
		var char2, prevChar2 rune
		leftCharCost := i
		aboveCharCost := i
		nextTransCost := 0
		for j := 0; j < len2; j++ {
			thisTransCost := nextTransCost
			nextTransCost = prevChar1Costs[j]
			prevChar1Costs[j] = currentCost
			currentCost = leftCharCost
			leftCharCost = char1Costs[j]
			prevChar2 = char2
			char2 = runeStr2[start+j]
			if char1 != char2 {
				// Substitution
				if aboveCharCost < currentCost {
					currentCost = aboveCharCost // Deletion
				}
				if leftCharCost < currentCost {
					currentCost = leftCharCost // Insertion
				}
				currentCost++
				if i != 0 && j != 0 && char1 == prevChar2 && prevChar1 == char2 && thisTransCost+1 < currentCost {
					currentCost = thisTransCost + 1 // Transposition
				}
			}
			char1Costs[j] = currentCost
			aboveCharCost = currentCost
		}
	}
	return currentCost
}

// distanceWithMax is the internal implementation of the core Damerau-Levenshtein, optimal string alignment algorithm that accepts a maxDistance.
func distanceWithMax(runeStr1, runeStr2 []rune, len1, len2, start, maxDistance int, char1Costs, prevChar1Costs []int) int {
	// Initialize the cost arrays
	for j := 0; j < maxDistance; j++ {
		char1Costs[j] = j + 1
	}
	for j := maxDistance; j < len2; j++ {
		char1Costs[j] = maxDistance + 1
	}
	lenDiff := len2 - len1
	jStartOffset := maxDistance - lenDiff
	jStart := 0
	jEnd := maxDistance
	var char1, prevChar1 rune
	var currentCost int
	for i := 0; i < len1; i++ {
		prevChar1 = char1
		char1 = runeStr1[start+i]
		var char2, prevChar2 rune
		leftCharCost := i
		aboveCharCost := i
		nextTransCost := 0
		// Adjust window
		if i > jStartOffset {
			jStart++
		}
		if jEnd < len2 {
			jEnd++
		}
		for j := jStart; j < jEnd; j++ {
			thisTransCost := nextTransCost
			nextTransCost = prevChar1Costs[j]
			prevChar1Costs[j] = currentCost
			currentCost = leftCharCost
			leftCharCost = char1Costs[j]
			prevChar2 = char2
			char2 = runeStr2[start+j]
			if char1 != char2 {
				// Substitution
				if aboveCharCost < currentCost {
					currentCost = aboveCharCost // Deletion
				}
				if leftCharCost < currentCost {
					currentCost = leftCharCost // Insertion
				}
				currentCost++
				if i != 0 && j != 0 && char1 == prevChar2 && prevChar1 == char2 && thisTransCost+1 < currentCost {
					currentCost = thisTransCost + 1 // Transposition
				}
			}
			char1Costs[j] = currentCost
			aboveCharCost = currentCost
		}
		if char1Costs[i+lenDiff] > maxDistance {
			return -1
		}
	}
	if currentCost <= maxDistance {
		return currentCost
	}
	return -1
}

// nullDistanceResults handles null or empty string cases for Distance function.
func nullDistanceResults(string1, string2 string, maxDistance int) int {
	if string1 == string2 {
		return 0
	}
	s1 := len([]rune(string1))
	s2 := len([]rune(string2))
	distance := s1
	if s2 > s1 {
		distance = s2
	}
	if distance > maxDistance {
		return -1
	}
	return distance
}

// prefixSuffixPrep identifies common prefix and suffix between the strings to optimize the computation.
func prefixSuffixPrep(runeStr1, runeStr2 []rune) (len1, len2, start int) {
	len1 = len(runeStr1)
	len2 = len(runeStr2)
	start = 0

	// Remove common prefix
	for start < len1 && start < len2 && runeStr1[start] == runeStr2[start] {
		start++
	}
	len1 -= start
	len2 -= start

	// Remove common suffix
	for len1 > 0 && len2 > 0 && runeStr1[start+len1-1] == runeStr2[start+len2-1] {
		len1--
		len2--
	}
	return len1, len2, start
}

// comparer is a struct to compare distances using DamerauOSA.
type comparer struct {
	damerauOSA *DamerauOSA
}

// NewDistanceComparer creates a new instance of comparer.
func NewDistanceComparer() *comparer {
	return &comparer{
		damerauOSA: NewDamerauOSA(),
	}
}

// Compare computes the edit distance between two strings with a maximum distance.
func (dc *comparer) Compare(string1, string2 string, maxDistance int) int {
	return dc.damerauOSA.Distance(string1, string2, maxDistance)
}
