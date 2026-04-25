package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadDefaultsForEmptyPath(t *testing.T) {
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := Config{HTTPPort: DefaultHTTPPort, DatabasePath: DefaultDatabasePath, PackSizes: DefaultPackSizes}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLoadDefaultsForMissingFile(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := Config{HTTPPort: DefaultHTTPPort, DatabasePath: DefaultDatabasePath, PackSizes: DefaultPackSizes}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLoadReturnsOpenError(t *testing.T) {
	_, err := Load("invalid\x00path")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadValidOverride(t *testing.T) {
	path := writeConfigFile(t, `{"http_port":"9090","database_path":"custom.db","pack_sizes":[500,250,1000]}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := Config{HTTPPort: "9090", DatabasePath: "custom.db", PackSizes: []int{250, 500, 1000}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLoadUsesDefaultsForEmptyOverrideValues(t *testing.T) {
	path := writeConfigFile(t, `{"http_port":"","pack_sizes":[]}`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := Config{HTTPPort: DefaultHTTPPort, DatabasePath: DefaultDatabasePath, PackSizes: DefaultPackSizes}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestLoadReturnsDecodeError(t *testing.T) {
	path := writeConfigFile(t, `{`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadReturnsValidationError(t *testing.T) {
	path := writeConfigFile(t, `{"pack_sizes":[250,-1]}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "pack sizes") {
		t.Fatalf("got error %q, want pack size error", err.Error())
	}
}

func TestValidate(t *testing.T) {
	if err := validate(Config{HTTPPort: "8080", DatabasePath: "data.db", PackSizes: []int{250}}); err != nil {
		t.Fatalf("validate returned error: %v", err)
	}

	if err := validate(Config{HTTPPort: "", DatabasePath: "data.db", PackSizes: []int{250}}); err == nil {
		t.Fatal("expected missing port error")
	}

	if err := validate(Config{HTTPPort: "8080", DatabasePath: "", PackSizes: []int{250}}); err == nil {
		t.Fatal("expected missing database path error")
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}
