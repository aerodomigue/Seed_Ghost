package engine

import (
	"testing"
)

func TestCalculateUploadDelta_ZeroLeechers(t *testing.T) {
	cfg := DefaultRatioConfig()
	delta := CalculateUploadDelta(cfg, 0, 1800, 0)
	if delta != 0 {
		t.Errorf("expected 0 delta for 0 leechers, got %d", delta)
	}
}

func TestCalculateUploadDelta_OneLeecher(t *testing.T) {
	cfg := DefaultRatioConfig()
	// With 1 leecher, should use min speed (50 KB/s)
	// 50 * 1024 * 1800 = 92160000 bytes ± 15%
	for i := 0; i < 100; i++ {
		delta := CalculateUploadDelta(cfg, 1, 1800, 0)
		if delta <= 0 {
			t.Errorf("expected positive delta for 1 leecher, got %d", delta)
		}
		// Should be roughly 50KB/s * 1800s = ~90MB, within ±15% + rounding
		minExpected := int64(50 * 1024 * 1800 * 0.84)
		maxExpected := int64(50 * 1024 * 1800 * 1.16)
		if delta < minExpected || delta > maxExpected {
			t.Errorf("delta %d outside expected range [%d, %d]", delta, minExpected, maxExpected)
		}
	}
}

func TestCalculateUploadDelta_MidLeechers(t *testing.T) {
	cfg := DefaultRatioConfig()
	delta := CalculateUploadDelta(cfg, 15, 1800, 0)
	if delta <= 0 {
		t.Errorf("expected positive delta for 15 leechers, got %d", delta)
	}
	// Should be between min and max speed
	minDelta := CalculateUploadDelta(RatioConfig{MinSpeedKBs: 50, MaxSpeedKBs: 50}, 15, 1800, 0)
	maxDelta := CalculateUploadDelta(RatioConfig{MinSpeedKBs: 5000, MaxSpeedKBs: 5000}, 15, 1800, 0)
	// Just verify it's in a reasonable range (with jitter, exact comparison is tricky)
	if delta < minDelta/2 || delta > maxDelta*2 {
		t.Errorf("delta %d seems unreasonable for 15 leechers", delta)
	}
}

func TestCalculateUploadDelta_MaxLeechers(t *testing.T) {
	cfg := DefaultRatioConfig()
	delta := CalculateUploadDelta(cfg, 50, 1800, 0)
	if delta <= 0 {
		t.Errorf("expected positive delta for 50 leechers, got %d", delta)
	}
	// Should be roughly max speed
	minExpected := int64(5000 * 1024 * 1800 * 0.84)
	maxExpected := int64(5000 * 1024 * 1800 * 1.16)
	if delta < minExpected || delta > maxExpected {
		t.Errorf("delta %d outside expected range [%d, %d]", delta, minExpected, maxExpected)
	}
}

func TestCalculateUploadDelta_GradualProgression(t *testing.T) {
	cfg := DefaultRatioConfig()
	// With a small previous delta, should not jump more than 2x
	previousDelta := int64(16384) // 1 block
	delta := CalculateUploadDelta(cfg, 50, 1800, previousDelta)
	maxAllowed := previousDelta * 2
	if delta > maxAllowed {
		t.Errorf("delta %d exceeds 2x previous %d (max %d)", delta, previousDelta, maxAllowed)
	}
}

func TestCalculateUploadDelta_BlockAlignment(t *testing.T) {
	cfg := DefaultRatioConfig()
	for i := 0; i < 100; i++ {
		delta := CalculateUploadDelta(cfg, 5, 1800, 0)
		if delta%16384 != 0 {
			t.Errorf("delta %d not aligned to 16KB blocks", delta)
		}
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
