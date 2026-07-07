// freeze.go contains the logic that resolves which existing (before) slide
// each frozen (after) slide corresponds to.
package deck

import "slices"

// frozenMatchFloor is the minimum alignment score granted to a frozen after
// slide against any before slide. Frozen slides are expected to have diverged
// from their markdown source (that is often why they are frozen), so content
// similarity alone cannot identify their counterpart. The floor lets the
// order-preserving alignment adopt a counterpart by relative position, while
// staying below genuine content matches (e.g. layout+title scores 130) so
// that non-frozen slides win their anchors first.
const frozenMatchFloor = 100

// resolveFrozenSlides determines which before slide each frozen after slide
// corresponds to. It returns a map with after index as key and before index
// as value. Frozen after slides without a counterpart are absent from the map.
//
// Frozen slides cannot be matched by content similarity because their actual
// content may have diverged from the markdown source. Instead, non-frozen
// slides act as anchors through their content similarity, and frozen slides
// are matched by relative order between those anchors using an
// order-preserving alignment (Needleman-Wunsch style). A second pass matches
// the remaining frozen slides against unmatched before slides by similarity,
// which covers frozen slides moved across anchors.
func resolveFrozenSlides(before, after Slides) map[int]int {
	resolved := make(map[int]int)
	if len(before) == 0 || !slices.ContainsFunc(after, func(s *Slide) bool { return s.Freeze }) {
		return resolved
	}

	m := len(before)
	n := len(after)
	// dp[i][j] is the best alignment score between before[:i] and after[:j].
	// Gaps (unmatched slides on either side) carry no penalty.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			dp[i][j] = max(
				dp[i-1][j-1]+frozenAlignScore(before[i-1], after[j-1]),
				dp[i-1][j],
				dp[i][j-1],
			)
		}
	}

	// Backtrack, preferring matches over gaps so that frozen slides adopt a
	// counterpart whenever doing so does not lower the total score.
	matchedBefore := make([]bool, m)
	i, j := m, n
	for i > 0 && j > 0 {
		score := frozenAlignScore(before[i-1], after[j-1])
		switch {
		case score > 0 && dp[i][j] == dp[i-1][j-1]+score:
			if after[j-1].Freeze {
				resolved[j-1] = i - 1
			}
			matchedBefore[i-1] = true
			i--
			j--
		case dp[i][j] == dp[i-1][j]:
			i--
		default:
			j--
		}
	}

	// Second pass: frozen slides that the order-preserving alignment could
	// not match (e.g. frozen slides moved across anchors) adopt the most
	// similar unmatched before slide, as long as the content still resembles
	// the markdown source.
	for j, afterSlide := range after {
		if !afterSlide.Freeze {
			continue
		}
		if _, ok := resolved[j]; ok {
			continue
		}
		bestIdx := -1
		bestScore := 0
		for i, beforeSlide := range before {
			if matchedBefore[i] {
				continue
			}
			if score := getSimilarity(beforeSlide, afterSlide); score > bestScore {
				bestIdx = i
				bestScore = score
			}
		}
		if bestIdx >= 0 && bestScore >= frozenMatchFloor {
			resolved[j] = bestIdx
			matchedBefore[bestIdx] = true
		}
	}

	return resolved
}

// frozenAlignScore is the alignment score used by resolveFrozenSlides.
func frozenAlignScore(beforeSlide, afterSlide *Slide) int {
	score := getSimilarity(beforeSlide, afterSlide)
	if afterSlide.Freeze {
		return max(score, frozenMatchFloor)
	}
	return score
}
