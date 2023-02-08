// Copyright 2022 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"emperror.dev/errors"
	"github.com/rs/zerolog"

	"fybrik.io/fybrik/pkg/datapath"
	"fybrik.io/fybrik/pkg/environment"
	"fybrik.io/fybrik/pkg/logging"
	"fybrik.io/fybrik/pkg/optimizer"
)

// find an optimal solution for a data plane (given the optimization strategy),
// which also satisfies governance and admin policies,
func solve(env *datapath.Environment, datasets []datapath.DataInfo, log *zerolog.Logger) ([]datapath.Solution, error) {
	cspPath := environment.GetCSPPath()
	if environment.UseCSP() && cspPath != "" { // If a CSP solver is configured, use it to find a solution for all data paths at once
		cspOptimizer := optimizer.NewOptimizer(env, datasets, cspPath, log)
		solution, err := cspOptimizer.Solve()
		if err == nil {
			if len(solution) > 0 { // solver found a solution
				return solution, nil
			}
			// solver returned UNSAT
			msg := "Data plane cannot be constructed given the deployed modules and the active restrictions"
			log.Error().Msg(msg)
			logging.LogStructure("Data Items Context", datasets, log, zerolog.TraceLevel, true, true)
			logging.LogStructure("Module Map", env.Modules, log, zerolog.TraceLevel, true, true)
			return nil, errors.New(msg)
		}

		// CSP failed for an unknown reason:issue an error and fallback to finding a non-optimized solution
		msg := "Error solving CSP. Fybrik will now search for a solution without considering optimization goals."
		log.Error().Err(err).Msg(msg)
	}

	// FIXME: Do we warn the user if optimization goals are present but no CSP engine is set?
	// No solution from CSP: use PathBuilder to build solutions. A datapath is built separately for every dataset
	solutions := []datapath.Solution{}
	for i := range datasets {
		pathBuilder := PathBuilder{Log: log, Env: env, Asset: &datasets[i]}
		solution, err := pathBuilder.solve()
		if err != nil {
			return solutions, err
		}
		solutions = append(solutions, solution)
	}
	return solutions, nil
}
