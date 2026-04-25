package packing

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrInvalidOrderSize = errors.New("order size must be greater than 0")
	ErrOrderTooLarge    = errors.New("order size exceeds the calculation limit")
	ErrNoPackSizes      = errors.New("at least one pack size is required")
	ErrInvalidPackSize  = errors.New("pack sizes must be positive")
)

const MaxOrderSize = 1_000_000

type PackLine struct {
	Size     int `json:"size"`
	Quantity int `json:"quantity"`
}

type Plan struct {
	ItemsOrdered int        `json:"items_ordered"`
	ItemsShipped int        `json:"items_shipped"`
	ItemsOver    int        `json:"items_over"`
	TotalPacks   int        `json:"total_packs"`
	Packs        []PackLine `json:"packs"`
}

type candidate struct {
	packCount int
	prevTotal int
	packSize  int
	reachable bool
}

func Calculate(packSizes []int, itemsOrdered int) (Plan, error) {
	if itemsOrdered <= 0 {
		return Plan{}, ErrInvalidOrderSize
	}
	if itemsOrdered > MaxOrderSize {
		return Plan{}, ErrOrderTooLarge
	}

	packs, err := normalizePackSizes(packSizes)
	if err != nil {
		return Plan{}, err
	}

	maxPack := packs[len(packs)-1]
	limit := itemsOrdered + maxPack
	best := make([]candidate, limit+1)
	best[0] = candidate{reachable: true}

	// Each reachable total keeps the fewest packs for that exact total.
	// Scanning upward from the order size later preserves the challenge priority:
	// minimum shipped items first, then minimum pack count for that total.
	for total := 1; total <= limit; total++ {
		for _, packSize := range packs {
			prev := total - packSize
			if prev < 0 || !best[prev].reachable {
				continue
			}

			packCount := best[prev].packCount + 1
			if !best[total].reachable || packCount < best[total].packCount {
				best[total] = candidate{
					packCount: packCount,
					prevTotal: prev,
					packSize:  packSize,
					reachable: true,
				}
			}
		}
	}

	for total := itemsOrdered; ; total++ {
		if best[total].reachable {
			return buildPlan(best, total, itemsOrdered), nil
		}
	}
}

func normalizePackSizes(packSizes []int) ([]int, error) {
	if len(packSizes) == 0 {
		return nil, ErrNoPackSizes
	}

	normalized := slices.Clone(packSizes)
	slices.Sort(normalized)
	normalized = slices.Compact(normalized)

	for _, size := range normalized {
		if size <= 0 {
			return nil, fmt.Errorf("%w: %d", ErrInvalidPackSize, size)
		}
	}

	return normalized, nil
}

func buildPlan(best []candidate, total int, itemsOrdered int) Plan {
	quantities := map[int]int{}
	for current := total; current > 0; current = best[current].prevTotal {
		quantities[best[current].packSize]++
	}

	lines := make([]PackLine, 0, len(quantities))
	totalPacks := 0
	for size, quantity := range quantities {
		lines = append(lines, PackLine{Size: size, Quantity: quantity})
		totalPacks += quantity
	}

	slices.SortFunc(lines, func(a, b PackLine) int {
		return b.Size - a.Size
	})

	return Plan{
		ItemsOrdered: itemsOrdered,
		ItemsShipped: total,
		ItemsOver:    total - itemsOrdered,
		TotalPacks:   totalPacks,
		Packs:        lines,
	}
}
