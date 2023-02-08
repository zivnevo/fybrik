// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package optimizer

import (
	"os"
	"testing"

	"fybrik.io/fybrik/pkg/datapath"
	"fybrik.io/fybrik/pkg/logging"
)

var testLog = logging.LogInit("Optimizer", "Test")

func TestOptimizer(t *testing.T) {
	env := getTestEnv()
	opt := NewOptimizer(env, []datapath.DataInfo{*getDataInfo(env)}, os.Getenv("CSP_PATH"), &testLog)
	solution, err := opt.Solve()
	if err != nil {
		t.Fatalf("Failed solving constraint problem: %v", err)
	}
	if len(solution) == 0 {
		t.Error("Got no solution (UNSAT) while expecting one")
	}

	solutionLen := len(solution[0].DataPath)
	if solutionLen < 2 {
		t.Errorf("Solution is too short: %d", solutionLen)
	} else if solutionLen > 3 {
		t.Errorf("Solution is too long: %d", solutionLen)
	}
	for _, edge := range solution[0].DataPath {
		t.Log(edge)
	}
}
