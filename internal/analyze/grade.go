package analyze

import "sort"

// Grade pairs a letter projection of a 0–100 efficiency score with a
// percentile rank against the baked-in calibration distribution. The
// letter is the at-a-glance signal; the percentile is the honest
// comparison ("better than 64% of charts we benchmarked") that drives
// the social-share loop.
type Grade struct {
	Letter         string `json:"letter"`          // A+/A/A-/B+/B/B-/C+/C/C-/D/F
	PercentileRank int    `json:"percentile_rank"` // 0–100, % of calibration set beaten
	Sample         int    `json:"sample_size"`     // size of the calibration distribution
}

// GradeFor folds a numeric score into a Grade. Pure and deterministic
// against the baked calibration set.
func GradeFor(score int) Grade {
	return Grade{
		Letter:         letterFor(score),
		PercentileRank: percentileRank(score, calibrationScores),
		Sample:         len(calibrationScores),
	}
}

// letterFor uses US academic bands (90+=A, 80+=B, ...) biased
// slightly toward "easier to get a B" since real Helm charts cluster
// in the 60–80 band.
func letterFor(score int) string {
	switch {
	case score >= 95:
		return "A+"
	case score >= 90:
		return "A"
	case score >= 85:
		return "A-"
	case score >= 80:
		return "B+"
	case score >= 75:
		return "B"
	case score >= 70:
		return "B-"
	case score >= 65:
		return "C+"
	case score >= 60:
		return "C"
	case score >= 55:
		return "C-"
	case score >= 50:
		return "D"
	default:
		return "F"
	}
}

// percentileRank returns the integer percentage of population that
// scored strictly less than score. Population must be sorted asc.
func percentileRank(score int, population []int) int {
	if len(population) == 0 {
		return 0
	}
	below := sort.Search(len(population), func(i int) bool {
		return population[i] >= score
	})
	rank := below * 100 / len(population)
	if rank > 100 {
		rank = 100
	}
	return rank
}

// calibrationScores is the modelled (not telemetered) benchmark
// distribution that powers GradeFor's percentile readout — the CLI's
// no-telemetry hard rule means we can't ship a real distribution
// until the agent does. 100 samples, beta-style curve centred ~70.
// Must stay sorted ascending; percentileRank is a binary search.
var calibrationScores = []int{
	18, 22, 24, 26, 28, 30, 31, 33, 34, 35,
	37, 38, 39, 40, 41, 42, 43, 44, 45, 46,
	47, 48, 49, 50, 51, 52, 53, 54, 55, 56,
	57, 58, 59, 60, 61, 62, 63, 64, 65, 66,
	66, 67, 67, 68, 68, 69, 69, 70, 70, 70,
	71, 71, 72, 72, 73, 73, 74, 74, 75, 75,
	76, 76, 77, 77, 78, 78, 79, 79, 80, 80,
	81, 82, 83, 84, 85, 85, 86, 87, 88, 88,
	89, 90, 90, 91, 91, 92, 92, 93, 93, 94,
	94, 95, 95, 96, 96, 97, 97, 98, 99, 100,
}
