package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpenCreatesDirectoryAndMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "packs.db")

	repository, err := Open(context.Background(), path)
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	defer repository.Close()

	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("expected data directory: %v", err)
	}

	if err := repository.SeedPackSizesIfEmpty(context.Background(), []int{250}); err != nil {
		t.Fatalf("SeedPackSizesIfEmpty returned error: %v", err)
	}
}

func TestOpenReturnsDirectoryError(t *testing.T) {
	restore := replaceOpenDependencies(t)
	defer restore()

	sentinel := errors.New("mkdir failed")
	mkdirAll = func(string, os.FileMode) error {
		return sentinel
	}

	_, err := Open(context.Background(), "packs.db")
	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
}

func TestOpenReturnsDatabaseOpenError(t *testing.T) {
	restore := replaceOpenDependencies(t)
	defer restore()

	sentinel := errors.New("open failed")
	openDB = func(string, string) (*sql.DB, error) {
		return nil, sentinel
	}

	_, err := Open(context.Background(), "packs.db")
	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
}

func TestOpenReturnsMigrationError(t *testing.T) {
	restore := replaceOpenDependencies(t)
	defer restore()

	sentinel := errors.New("migrate failed")
	runMigrate = func(context.Context, *Repository) error {
		return sentinel
	}

	_, err := Open(context.Background(), filepath.Join(t.TempDir(), "packs.db"))
	if !errors.Is(err, sentinel) {
		t.Fatalf("got error %v, want %v", err, sentinel)
	}
}

func TestRepositoryGetPackSizes(t *testing.T) {
	repository := openTestRepository(t)
	replacePackSizes(t, repository, []int{500, 250})

	got, err := repository.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("GetPackSizes returned error: %v", err)
	}

	want := []int{250, 500}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestRepositoryGetPackSizesQueryError(t *testing.T) {
	repository := openTestRepository(t)
	if err := repository.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	_, err := repository.GetPackSizes(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRepositoryGetPackSizesScanError(t *testing.T) {
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "bad-schema.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if _, err := db.ExecContext(context.Background(), `CREATE TABLE pack_sizes(size TEXT); INSERT INTO pack_sizes(size) VALUES ('bad');`); err != nil {
		t.Fatalf("create bad schema: %v", err)
	}

	repository := NewRepository(db)
	_, err = repository.GetPackSizes(context.Background())
	if err == nil {
		t.Fatal("expected scan error")
	}
}

func TestRepositoryReplacePackSizes(t *testing.T) {
	repository := openTestRepository(t)

	got, err := repository.ReplacePackSizes(context.Background(), []int{250, 500})
	if err != nil {
		t.Fatalf("ReplacePackSizes returned error: %v", err)
	}

	want := []int{250, 500}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}

	stored, err := repository.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("GetPackSizes returned error: %v", err)
	}
	if !reflect.DeepEqual(stored, want) {
		t.Fatalf("stored %#v, want %#v", stored, want)
	}
}

func TestRepositoryReplacePackSizesBeginError(t *testing.T) {
	repository := openTestRepository(t)
	if err := repository.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	_, err := repository.ReplacePackSizes(context.Background(), []int{250})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRepositoryReplacePackSizesDeleteError(t *testing.T) {
	repository := openTestRepository(t)
	if _, err := repository.db.ExecContext(context.Background(), `DROP TABLE pack_sizes`); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	_, err := repository.ReplacePackSizes(context.Background(), []int{250})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRepositoryReplacePackSizesInsertErrorRollsBack(t *testing.T) {
	repository := openTestRepository(t)
	replacePackSizes(t, repository, []int{250})

	_, err := repository.ReplacePackSizes(context.Background(), []int{500, -1})
	if err == nil {
		t.Fatal("expected error")
	}

	got, err := repository.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("GetPackSizes returned error: %v", err)
	}
	want := []int{250}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestRepositorySeedPackSizesIfEmpty(t *testing.T) {
	repository := openTestRepository(t)

	if err := repository.SeedPackSizesIfEmpty(context.Background(), []int{250, 500}); err != nil {
		t.Fatalf("SeedPackSizesIfEmpty returned error: %v", err)
	}
	if err := repository.SeedPackSizesIfEmpty(context.Background(), []int{1000}); err != nil {
		t.Fatalf("SeedPackSizesIfEmpty returned error: %v", err)
	}

	got, err := repository.GetPackSizes(context.Background())
	if err != nil {
		t.Fatalf("GetPackSizes returned error: %v", err)
	}
	want := []int{250, 500}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestRepositorySeedPackSizesIfEmptyCountError(t *testing.T) {
	repository := openTestRepository(t)
	if _, err := repository.db.ExecContext(context.Background(), `DROP TABLE pack_sizes`); err != nil {
		t.Fatalf("drop table: %v", err)
	}

	err := repository.SeedPackSizesIfEmpty(context.Background(), []int{250})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRepositoryClose(t *testing.T) {
	repository := openTestRepository(t)
	if err := repository.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
}

func replaceOpenDependencies(t *testing.T) func() {
	t.Helper()

	originalMkdirAll := mkdirAll
	originalOpenDB := openDB
	originalRunMigrate := runMigrate

	return func() {
		mkdirAll = originalMkdirAll
		openDB = originalOpenDB
		runMigrate = originalRunMigrate
	}
}

func openTestRepository(t *testing.T) *Repository {
	t.Helper()

	repository, err := Open(context.Background(), filepath.Join(t.TempDir(), "packs.db"))
	if err != nil {
		t.Fatalf("Open returned error: %v", err)
	}
	t.Cleanup(func() {
		_ = repository.Close()
	})
	return repository
}

func replacePackSizes(t *testing.T, repository *Repository, sizes []int) {
	t.Helper()
	if _, err := repository.ReplacePackSizes(context.Background(), sizes); err != nil {
		t.Fatalf("ReplacePackSizes returned error: %v", err)
	}
}
