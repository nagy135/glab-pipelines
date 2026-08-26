package main

type limitOption struct {
	Value       int
	Description string
}

var limitOptions = []limitOption{
	{Value: 5, Description: "compact"},
	{Value: 10, Description: "default"},
	{Value: 20, Description: "more history"},
	{Value: 30, Description: "extended history"},
	{Value: 50, Description: "large history"},
	{Value: 100, Description: "maximum history"},
}

func limitIndex(limit int) int {
	closest := 0
	closestDistance := intDistance(limit, limitOptions[0].Value)
	for i, option := range limitOptions {
		if option.Value == limit {
			return i
		}
		distance := intDistance(limit, option.Value)
		if distance < closestDistance {
			closest = i
			closestDistance = distance
		}
	}
	return closest
}

func intDistance(a, b int) int {
	if a < b {
		return b - a
	}
	return a - b
}
