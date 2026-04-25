package packs

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"pack-shipping-calculator/backend/internal/packing"
)

func TestNormalizePackSizes(t *testing.T) {
	got, err := NormalizePackSizes([]int{500, 250, 500, 1000})
	if err != nil {
		t.Fatalf("NormalizePackSizes returned error: %v", err)
	}

	want := []int{250, 500, 1000}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestNormalizePackSizesValidation(t *testing.T) {
	tests := []struct {
		name  string
		sizes []int
	}{
		{name: "empty", sizes: nil},
		{name: "zero", sizes: []int{250, 0}},
		{name: "negative", sizes: []int{-1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NormalizePackSizes(tt.sizes)
			if !errors.Is(err, ErrInvalidPackSizes) {
				t.Fatalf("got error %v, want %v", err, ErrInvalidPackSizes)
			}
		})
	}
}

func TestServiceGetPackSizes(t *testing.T) {
	service := NewService(&fakeRepository{sizes: []int{500, 250, 500}})

	got, err := service.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("GetPackSizes returned error: %v", err)
	}

	want := []int{250, 500}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestServiceGetPackSizesRepositoryError(t *testing.T) {
	sentinel := errors.New("get failed")
	service := NewService(&fakeRepository{getErr: sentinel})

	_, err := service.GetPackSizes(context.Background())
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("got error %v, want repository error", err)
	}
}

func TestServiceGetPackSizesInvalidRepositoryData(t *testing.T) {
	service := NewService(&fakeRepository{sizes: []int{0}})

	_, err := service.GetPackSizes(context.Background())
	if !errors.Is(err, ErrInvalidPackSizes) {
		t.Fatalf("got error %v, want invalid pack sizes", err)
	}
}

func TestServiceReplacePackSizes(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	got, err := service.ReplacePackSizes(context.Background(), []int{500, 250, 500})
	if err != nil {
		t.Fatalf("ReplacePackSizes returned error: %v", err)
	}

	want := []int{250, 500}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(repository.replacedSizes, want) {
		t.Fatalf("got repository sizes %#v, want %#v", repository.replacedSizes, want)
	}
}

func TestServiceReplacePackSizesValidationPreventsRepositoryWrite(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	_, err := service.ReplacePackSizes(context.Background(), []int{0})
	if !errors.Is(err, ErrInvalidPackSizes) {
		t.Fatalf("got error %v, want invalid pack sizes", err)
	}
	if repository.replaceCalled {
		t.Fatal("repository should not be called")
	}
}

func TestServiceReplacePackSizesRepositoryError(t *testing.T) {
	sentinel := errors.New("replace failed")
	service := NewService(&fakeRepository{replaceErr: sentinel})

	_, err := service.ReplacePackSizes(context.Background(), []int{250})
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("got error %v, want repository error", err)
	}
}

func TestServiceReplacePackSizesInvalidRepositoryResponse(t *testing.T) {
	service := NewService(&fakeRepository{replaceResponse: []int{0}})

	_, err := service.ReplacePackSizes(context.Background(), []int{250})
	if !errors.Is(err, ErrInvalidPackSizes) {
		t.Fatalf("got error %v, want invalid pack sizes", err)
	}
}

func TestServiceSeedPackSizesIfEmpty(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	if err := service.SeedPackSizesIfEmpty(context.Background(), []int{500, 250, 500}); err != nil {
		t.Fatalf("SeedPackSizesIfEmpty returned error: %v", err)
	}

	want := []int{250, 500}
	if !reflect.DeepEqual(repository.seededSizes, want) {
		t.Fatalf("got seeded sizes %#v, want %#v", repository.seededSizes, want)
	}
}

func TestServiceSeedPackSizesIfEmptyValidation(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	err := service.SeedPackSizesIfEmpty(context.Background(), nil)
	if !errors.Is(err, ErrInvalidPackSizes) {
		t.Fatalf("got error %v, want invalid pack sizes", err)
	}
	if repository.seedCalled {
		t.Fatal("repository should not be called")
	}
}

func TestServiceSeedPackSizesIfEmptyRepositoryError(t *testing.T) {
	sentinel := errors.New("seed failed")
	service := NewService(&fakeRepository{seedErr: sentinel})

	err := service.SeedPackSizesIfEmpty(context.Background(), []int{250})
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("got error %v, want repository error", err)
	}
}

func TestServiceCalculateOrder(t *testing.T) {
	service := NewService(&fakeRepository{sizes: []int{250, 500, 1000, 2000, 5000}})

	got, err := service.CalculateOrder(context.Background(), 251)
	if err != nil {
		t.Fatalf("CalculateOrder returned error: %v", err)
	}

	want := packing.Plan{
		ItemsOrdered: 251,
		ItemsShipped: 500,
		ItemsOver:    249,
		TotalPacks:   1,
		Packs:        []packing.PackLine{{Size: 500, Quantity: 1}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestServiceCalculateOrderRepositoryError(t *testing.T) {
	service := NewService(&fakeRepository{getErr: errors.New("get failed")})

	_, err := service.CalculateOrder(context.Background(), 1)
	if !errors.Is(err, ErrRepository) {
		t.Fatalf("got error %v, want repository error", err)
	}
}

func TestServiceClose(t *testing.T) {
	sentinel := errors.New("close failed")
	service := NewService(&fakeRepository{closeErr: sentinel})

	err := service.Close()
	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
}

type fakeRepository struct {
	sizes           []int
	replacedSizes   []int
	replaceResponse []int
	seededSizes     []int
	getErr          error
	replaceErr      error
	seedErr         error
	closeErr        error
	replaceCalled   bool
	seedCalled      bool
}

func (r *fakeRepository) GetPackSizes(context.Context) ([]int, error) {
	return r.sizes, r.getErr
}

func (r *fakeRepository) ReplacePackSizes(_ context.Context, sizes []int) ([]int, error) {
	r.replaceCalled = true
	r.replacedSizes = sizes
	if r.replaceResponse != nil {
		return r.replaceResponse, r.replaceErr
	}
	return sizes, r.replaceErr
}

func (r *fakeRepository) SeedPackSizesIfEmpty(_ context.Context, sizes []int) error {
	r.seedCalled = true
	r.seededSizes = sizes
	return r.seedErr
}

func (r *fakeRepository) Close() error {
	return r.closeErr
}
