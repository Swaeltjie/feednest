package scorer

import (
	"testing"
	"time"
)

func TestCalculateScore(t *testing.T) {
	now := time.Now()

	score1 := CalculateScore(now.Add(-1*time.Hour), 0.8)
	score2 := CalculateScore(now.Add(-48*time.Hour), 0.1)

	if score1 <= score2 {
		t.Errorf("recent high-engagement article (%.2f) should score higher than old low-engagement (%.2f)", score1, score2)
	}
}

func TestRecencyScore(t *testing.T) {
	now := time.Now()
	score1h := recencyScore(now.Add(-1 * time.Hour))
	score24h := recencyScore(now.Add(-24 * time.Hour))
	score72h := recencyScore(now.Add(-72 * time.Hour))

	if score1h <= score24h || score24h <= score72h {
		t.Errorf("recency should decrease over time: 1h=%.2f, 24h=%.2f, 72h=%.2f", score1h, score24h, score72h)
	}
}

func TestRecencyScore_FutureDate(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	score := recencyScore(future)
	if score < 0.99 {
		t.Errorf("future date should have score ~1.0, got %.2f", score)
	}
}

func TestCalculateScore_ZeroEngagement(t *testing.T) {
	now := time.Now()
	score := CalculateScore(now, 0.0)
	if score <= 0 {
		t.Errorf("recent article with zero engagement should still have positive score, got %.2f", score)
	}
}
