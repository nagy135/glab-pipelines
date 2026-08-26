package main

import "time"

type refreshOption struct {
	Duration    time.Duration
	Description string
}

var refreshOptions = []refreshOption{
	{Duration: 5 * time.Second, Description: "very frequent"},
	{Duration: 10 * time.Second, Description: "frequent"},
	{Duration: 20 * time.Second, Description: "default"},
	{Duration: 30 * time.Second, Description: "balanced"},
	{Duration: time.Minute, Description: "low traffic"},
	{Duration: 5 * time.Minute, Description: "minimal traffic"},
}

func refreshIndex(refresh time.Duration) int {
	closest := 0
	closestDistance := durationDistance(refresh, refreshOptions[0].Duration)
	for i, option := range refreshOptions {
		if option.Duration == refresh {
			return i
		}
		distance := durationDistance(refresh, option.Duration)
		if distance < closestDistance {
			closest = i
			closestDistance = distance
		}
	}
	return closest
}

func durationDistance(a, b time.Duration) time.Duration {
	if a < b {
		return b - a
	}
	return a - b
}
