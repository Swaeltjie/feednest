package scorer

import (
	"math"
	"time"
)

const (
	recencyWeight    = 0.6
	engagementWeight = 0.4
	decayHalfLife    = 24.0 // hours
)

func CalculateScore(publishedAt time.Time, feedEngagement float64) float64 {
	recency := recencyScore(publishedAt)
	return (recencyWeight * recency) + (engagementWeight * feedEngagement)
}

func recencyScore(publishedAt time.Time) float64 {
	hoursAgo := time.Since(publishedAt).Hours()
	if hoursAgo < 0 {
		hoursAgo = 0
	}
	return math.Exp(-0.693 * hoursAgo / decayHalfLife) // 0.693 = ln(2)
}
