package packing

import (
	"errors"
	"reflect"
	"testing"
)

func TestCalculateChallengeExamples(t *testing.T) {
	tests := []struct {
		name         string
		packSizes    []int
		itemsOrdered int
		want         Plan
	}{
		{
			name:         "one item",
			packSizes:    []int{250, 500, 1000, 2000, 5000},
			itemsOrdered: 1,
			want: Plan{
				ItemsOrdered: 1,
				ItemsShipped: 250,
				ItemsOver:    249,
				TotalPacks:   1,
				Packs:        []PackLine{{Size: 250, Quantity: 1}},
			},
		},
		{
			name:         "exact smallest pack",
			packSizes:    []int{250, 500, 1000, 2000, 5000},
			itemsOrdered: 250,
			want: Plan{
				ItemsOrdered: 250,
				ItemsShipped: 250,
				ItemsOver:    0,
				TotalPacks:   1,
				Packs:        []PackLine{{Size: 250, Quantity: 1}},
			},
		},
		{
			name:         "one over smallest pack",
			packSizes:    []int{250, 500, 1000, 2000, 5000},
			itemsOrdered: 251,
			want: Plan{
				ItemsOrdered: 251,
				ItemsShipped: 500,
				ItemsOver:    249,
				TotalPacks:   1,
				Packs:        []PackLine{{Size: 500, Quantity: 1}},
			},
		},
		{
			name:         "minimum items beats fewer larger pack",
			packSizes:    []int{250, 500, 1000, 2000, 5000},
			itemsOrdered: 501,
			want: Plan{
				ItemsOrdered: 501,
				ItemsShipped: 750,
				ItemsOver:    249,
				TotalPacks:   2,
				Packs:        []PackLine{{Size: 500, Quantity: 1}, {Size: 250, Quantity: 1}},
			},
		},
		{
			name:         "large order",
			packSizes:    []int{250, 500, 1000, 2000, 5000},
			itemsOrdered: 12001,
			want: Plan{
				ItemsOrdered: 12001,
				ItemsShipped: 12250,
				ItemsOver:    249,
				TotalPacks:   4,
				Packs:        []PackLine{{Size: 5000, Quantity: 2}, {Size: 2000, Quantity: 1}, {Size: 250, Quantity: 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.packSizes, tt.itemsOrdered)
			if err != nil {
				t.Fatalf("Calculate returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCalculateWithVariablePackSizes(t *testing.T) {
	tests := []struct {
		name         string
		packSizes    []int
		itemsOrdered int
		want         Plan
	}{
		{
			name:         "custom sizes choose minimum shipped items over fewer packs",
			packSizes:    []int{5, 9, 20},
			itemsOrdered: 10,
			want: Plan{
				ItemsOrdered: 10,
				ItemsShipped: 10,
				ItemsOver:    0,
				TotalPacks:   2,
				Packs:        []PackLine{{Size: 5, Quantity: 2}},
			},
		},
		{
			name:         "custom sizes choose fewest packs when shipped items are equal",
			packSizes:    []int{4, 6, 10},
			itemsOrdered: 12,
			want: Plan{
				ItemsOrdered: 12,
				ItemsShipped: 12,
				ItemsOver:    0,
				TotalPacks:   2,
				Packs:        []PackLine{{Size: 6, Quantity: 2}},
			},
		},
		{
			name:         "custom sizes can combine non default packs",
			packSizes:    []int{3, 5},
			itemsOrdered: 7,
			want: Plan{
				ItemsOrdered: 7,
				ItemsShipped: 8,
				ItemsOver:    1,
				TotalPacks:   2,
				Packs:        []PackLine{{Size: 5, Quantity: 1}, {Size: 3, Quantity: 1}},
			},
		},
		{
			name:         "unsorted duplicated custom sizes are normalized",
			packSizes:    []int{9, 3, 3, 6},
			itemsOrdered: 14,
			want: Plan{
				ItemsOrdered: 14,
				ItemsShipped: 15,
				ItemsOver:    1,
				TotalPacks:   2,
				Packs:        []PackLine{{Size: 9, Quantity: 1}, {Size: 6, Quantity: 1}},
			},
		},
		{
			name:         "minimum shipped items still wins with awkward custom sizes",
			packSizes:    []int{7, 11, 25},
			itemsOrdered: 26,
			want: Plan{
				ItemsOrdered: 26,
				ItemsShipped: 28,
				ItemsOver:    2,
				TotalPacks:   4,
				Packs:        []PackLine{{Size: 7, Quantity: 4}},
			},
		},
		{
			name:         "runtime style pack sizes replace the defaults",
			packSizes:    []int{100, 400},
			itemsOrdered: 450,
			want: Plan{
				ItemsOrdered: 450,
				ItemsShipped: 500,
				ItemsOver:    50,
				TotalPacks:   2,
				Packs:        []PackLine{{Size: 400, Quantity: 1}, {Size: 100, Quantity: 1}},
			},
		},
		{
			name:         "non default exact larger pack beats multiple smaller packs",
			packSizes:    []int{25, 50, 100, 200},
			itemsOrdered: 176,
			want: Plan{
				ItemsOrdered: 176,
				ItemsShipped: 200,
				ItemsOver:    24,
				TotalPacks:   1,
				Packs:        []PackLine{{Size: 200, Quantity: 1}},
			},
		},
		{
			name:         "default sized unsorted duplicates are normalized",
			packSizes:    []int{500, 250, 500, 1000},
			itemsOrdered: 751,
			want: Plan{
				ItemsOrdered: 751,
				ItemsShipped: 1000,
				ItemsOver:    249,
				TotalPacks:   1,
				Packs:        []PackLine{{Size: 1000, Quantity: 1}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Calculate(tt.packSizes, tt.itemsOrdered)
			if err != nil {
				t.Fatalf("Calculate returned error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCalculateValidation(t *testing.T) {
	tests := []struct {
		name         string
		packSizes    []int
		itemsOrdered int
		wantErr      error
	}{
		{name: "invalid order", packSizes: []int{250}, itemsOrdered: 0, wantErr: ErrInvalidOrderSize},
		{name: "order too large", packSizes: []int{250}, itemsOrdered: MaxOrderSize + 1, wantErr: ErrOrderTooLarge},
		{name: "no packs", packSizes: nil, itemsOrdered: 1, wantErr: ErrNoPackSizes},
		{name: "invalid pack", packSizes: []int{250, 0}, itemsOrdered: 1, wantErr: ErrInvalidPackSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Calculate(tt.packSizes, tt.itemsOrdered)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want %v", err, tt.wantErr)
			}
		})
	}
}
