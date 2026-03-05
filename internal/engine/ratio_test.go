package engine

import (
	"testing"
)

func TestCalculateWeight_ZeroLeechers(t *testing.T) {
	w := CalculateWeight(0, 5)
	if w != 0 {
		t.Errorf("expected 0 weight for 0 leechers, got %f", w)
	}
}

func TestCalculateWeight_WithLeechers(t *testing.T) {
	// 10 leechers, 2 seeders
	// ratio = 10/12 = 0.833
	// weight = 0.833^2 * 100 * 10 = 694
	w := CalculateWeight(10, 2)
	if w < 690 || w > 700 {
		t.Errorf("expected weight ~694 for 10L/2S, got %f", w)
	}
}

func TestCalculateWeight_OnlyLeechers(t *testing.T) {
	// 5 leechers, 0 seeders → ratio = 1.0
	// weight = 1.0^2 * 100 * 5 = 500
	w := CalculateWeight(5, 0)
	if w != 500 {
		t.Errorf("expected weight 500 for 5L/0S, got %f", w)
	}
}

func TestCalculateWeight_MoreLeechersMoreWeight(t *testing.T) {
	w1 := CalculateWeight(2, 5)
	w2 := CalculateWeight(10, 5)
	w3 := CalculateWeight(50, 5)
	if w1 >= w2 || w2 >= w3 {
		t.Errorf("expected increasing weight with more leechers: %f, %f, %f", w1, w2, w3)
	}
}

func TestRandomGlobalSpeed(t *testing.T) {
	cfg := RatioConfig{MinSpeedKBs: 100, MaxSpeedKBs: 1000}
	minB := cfg.MinSpeedKBs * 1024
	maxB := cfg.MaxSpeedKBs * 1024
	for i := 0; i < 100; i++ {
		speed := RandomGlobalSpeed(cfg)
		if speed < minB || speed > maxB {
			t.Errorf("speed %f outside range [%f, %f]", speed, minB, maxB)
		}
	}
}

func TestRandomGlobalSpeed_EqualMinMax(t *testing.T) {
	cfg := RatioConfig{MinSpeedKBs: 500, MaxSpeedKBs: 500}
	speed := RandomGlobalSpeed(cfg)
	expected := 500.0 * 1024
	if speed != expected {
		t.Errorf("expected %f, got %f", expected, speed)
	}
}

func TestApplyJitter(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := ApplyJitter(1800)
		if result < 1620 || result > 1980 { // ±10%
			t.Errorf("jitter result %d outside ±10%% of 1800", result)
		}
	}
}

func TestApplyJitter_Zero(t *testing.T) {
	result := ApplyJitter(0)
	if result != 0 {
		t.Errorf("expected 0, got %d", result)
	}
}
