package engine

import (
	"math/rand"
)

const blockSize = 16384 // 16KB - standard BitTorrent block size

// RatioConfig holds the configuration for upload speed calculation.
type RatioConfig struct {
	MinSpeedKBs         float64 // Minimum global upload speed in KB/s
	MaxSpeedKBs         float64 // Maximum global upload speed in KB/s
	MinDownloadSpeedKBs float64 // Minimum per-session download speed in KB/s
	MaxDownloadSpeedKBs float64 // Maximum per-session download speed in KB/s
}

// DefaultRatioConfig returns a RatioConfig with sensible defaults.
func DefaultRatioConfig() RatioConfig {
	return RatioConfig{
		MinSpeedKBs:         50,
		MaxSpeedKBs:         5000,
		MinDownloadSpeedKBs: 100,
		MaxDownloadSpeedKBs: 10000,
	}
}

// RandomGlobalSpeed picks a random speed between min and max (in bytes/s).
// Called every ~20 minutes to refresh the global bandwidth, like JOAL.
func RandomGlobalSpeed(cfg RatioConfig) float64 {
	minB := cfg.MinSpeedKBs * 1024
	maxB := cfg.MaxSpeedKBs * 1024
	if minB >= maxB {
		return maxB
	}
	return minB + rand.Float64()*(maxB-minB)
}

// RandomDownloadSpeed picks a random per-session download speed between min and max (in bytes/s).
func RandomDownloadSpeed(cfg RatioConfig) float64 {
	minB := cfg.MinDownloadSpeedKBs * 1024
	maxB := cfg.MaxDownloadSpeedKBs * 1024
	if minB >= maxB {
		return maxB
	}
	return minB + rand.Float64()*(maxB-minB)
}

// CalculateWeight computes the bandwidth weight for a torrent based on its peers.
// Formula (from JOAL): weight = leechersRatio² × 100 × leechers
// where leechersRatio = leechers / (leechers + seeders).
// Returns 0 if no leechers (no upload when nobody is downloading).
func CalculateWeight(leechers, seeders int) float64 {
	if leechers <= 0 {
		return 0
	}
	total := float64(leechers + seeders)
	if total <= 0 {
		return 0
	}
	ratio := float64(leechers) / total
	return ratio * ratio * 100.0 * float64(leechers)
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
