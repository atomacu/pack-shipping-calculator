package packs

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"pack-shipping-calculator/backend/internal/packing"
)

var (
	ErrInvalidPackSizes = errors.New("pack sizes must contain at least one positive integer")
	ErrRepository       = errors.New("pack size repository error")
)

type Repository interface {
	GetPackSizes(context.Context) ([]int, error)
	ReplacePackSizes(context.Context, []int) ([]int, error)
	SeedPackSizesIfEmpty(context.Context, []int) error
	Close() error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) GetPackSizes(ctx context.Context) ([]int, error) {
	sizes, err := s.repository.GetPackSizes(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepository, err)
	}
	return NormalizePackSizes(sizes)
}

func (s *Service) ReplacePackSizes(ctx context.Context, sizes []int) ([]int, error) {
	normalized, err := NormalizePackSizes(sizes)
	if err != nil {
		return nil, err
	}

	replaced, err := s.repository.ReplacePackSizes(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRepository, err)
	}
	return NormalizePackSizes(replaced)
}

func (s *Service) SeedPackSizesIfEmpty(ctx context.Context, sizes []int) error {
	normalized, err := NormalizePackSizes(sizes)
	if err != nil {
		return err
	}

	if err := s.repository.SeedPackSizesIfEmpty(ctx, normalized); err != nil {
		return fmt.Errorf("%w: %v", ErrRepository, err)
	}
	return nil
}

func (s *Service) CalculateOrder(ctx context.Context, items int) (packing.Plan, error) {
	sizes, err := s.GetPackSizes(ctx)
	if err != nil {
		return packing.Plan{}, err
	}
	return packing.Calculate(sizes, items)
}

func (s *Service) Close() error {
	return s.repository.Close()
}

func NormalizePackSizes(sizes []int) ([]int, error) {
	if len(sizes) == 0 {
		return nil, ErrInvalidPackSizes
	}

	normalized := slices.Clone(sizes)
	slices.Sort(normalized)
	normalized = slices.Compact(normalized)

	for _, size := range normalized {
		if size <= 0 {
			return nil, ErrInvalidPackSizes
		}
	}

	return normalized, nil
}
