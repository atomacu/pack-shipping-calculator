package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestMainSuccess(t *testing.T) {
	restore := replaceMainDependencies(t)
	defer restore()

	runCalled := false
	runApp = func(ctx context.Context) error {
		if ctx == nil {
			t.Fatal("expected context")
		}
		runCalled = true
		return nil
	}
	exitProcess = func(code int) {
		t.Fatalf("exitProcess called with code %d", code)
	}

	main()

	if !runCalled {
		t.Fatal("expected runApp to be called")
	}
}

func TestMainFailure(t *testing.T) {
	restore := replaceMainDependencies(t)
	defer restore()

	sentinel := errors.New("run failed")
	var logged []any
	exitCode := 0

	runApp = func(context.Context) error {
		return sentinel
	}
	logError = func(values ...any) {
		logged = values
	}
	exitProcess = func(code int) {
		exitCode = code
	}

	main()

	if !reflect.DeepEqual(logged, []any{sentinel}) {
		t.Fatalf("got logged %#v, want sentinel error", logged)
	}
	if exitCode != 1 {
		t.Fatalf("got exit code %d, want 1", exitCode)
	}
}

func replaceMainDependencies(t *testing.T) func() {
	t.Helper()

	originalRunApp := runApp
	originalExitProcess := exitProcess
	originalLogError := logError

	return func() {
		runApp = originalRunApp
		exitProcess = originalExitProcess
		logError = originalLogError
	}
}
