package core

import (
	"errors"
	"slices"
	"testing"
)

func TestExecuteStagesStopsAfterError(t *testing.T) {
	want := errors.New("stop")
	var calls []int

	got, err := executeStages(0, "option",
		func(ctx int, option string) (int, error) {
			calls = append(calls, ctx)
			if option != "option" {
				t.Fatal("option was not passed to the stage")
			}
			return ctx + 1, nil
		},
		func(ctx int, _ string) (int, error) {
			calls = append(calls, ctx)
			return ctx + 1, want
		},
		func(ctx int, _ string) (int, error) {
			t.Fatal("stage after error was run")
			return ctx, nil
		},
	)

	if got != 2 || !errors.Is(err, want) || !slices.Equal(calls, []int{0, 1}) {
		t.Fatalf("executeStages() = (%d, %v), calls = %v; want (2, %v), [0 1]", got, err, calls, want)
	}
}
