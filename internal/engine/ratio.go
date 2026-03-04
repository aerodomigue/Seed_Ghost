package engine

import (
	"math"
	"math/rand"
)

const blockSize = 16384 // 16KB - standard BitTorrent block size

// RatioConfig holds the configuration for upload speed calculation.
type RatioConfig struct {
	MinSpeedKBs float64 // Minimum upload speed in KB/s (used with 1-2 leechers)
	MaxSpeedKBs float64 // Maximum upload speed in KB/s (used with 30+ leechers)
}

// DefaultRatioConfig returns a RatioConfig with sensible defaults.
func DefaultRatioConfig() RatioConfig {
	return RatioConfig{
		MinSpeedKBs: 50,
		MaxSpeedKBs: 5000,
	}
}

// CalculateUploadDelta calculates how many bytes to report as uploaded
// for a given announce interval, based on the number of leechers.
//
// Rules:
//   - 0 leechers: 0 bytes (never fake upload without receivers)
//   - 1-2 leechers: minimum speed
//   - 3-29 leechers: linear interpolation between min and max
//   - 30+ leechers: maximum speed
//   - ±15% random variation
//   - Gradual progression (no jumps > 2x previous delta)
//   - Round to 16KB blocks
func CalculateUploadDelta(cfg RatioConfig, leechers int, intervalSecs int, previousDelta int64) int64 {
	if leechers <= 0 || intervalSecs <= 0 {
		return 0
	}

	// Determine speed in KB/s based on leechers
	var speedKBs float64
	switch {
	case leechers <= 2:
		speedKBs = cfg.MinSpeedKBs
	case leechers >= 30:
		speedKBs = cfg.MaxSpeedKBs
	default:
		// Linear interpolation between min and max for 3-29 leechers
		ratio := float64(leechers-2) / float64(30-2)
		speedKBs = cfg.MinSpeedKBs + ratio*(cfg.MaxSpeedKBs-cfg.MinSpeedKBs)
	}

	// Calculate base delta in bytes
	baseDelta := speedKBs * 1024 * float64(intervalSecs)

	// Apply ±15% random variation
	jitter := 1.0 + (rand.Float64()*0.3 - 0.15) // [0.85, 1.15]
	delta := baseDelta * jitter

	// Gradual progression: cap at 2x previous delta (if previous > 0)
	if previousDelta > 0 {
		maxDelta := float64(previousDelta) * 2.0
		if delta > maxDelta {
			delta = maxDelta
		}
	}

	// Round to 16KB blocks
	blocks := int64(math.Round(delta / float64(blockSize)))
	if blocks <= 0 && leechers > 0 {
		blocks = 1 // At least 1 block if there are leechers
	}

	return blocks * blockSize
}

// ApplyJitter adds ±10% jitter to an interval in seconds.
func ApplyJitter(intervalSecs int) int {
	if intervalSecs <= 0 {
		return intervalSecs
	}
	jitter := 1.0 + (rand.Float64()*0.2 - 0.1) // [0.9, 1.1]
	result := int(float64(intervalSecs) * jitter)
	if result <= 0 {
		result = 1
	}
	return result
}
