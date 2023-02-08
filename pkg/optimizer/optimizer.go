// Copyright 2020 IBM Corp.
// SPDX-License-Identifier: Apache-2.0

/*
	This package is for finding an optimal data plane under constraints.
	Its main Optimizer class takes dataset and infrastructure metadata, restrictions and optimization goals.
	Optimizer.Solve() returns a valid and optimal data plane connecting a collection of datasets to a user workload
	(if such a data plane exists).

	All relevant data gets translated into a Constraint Satisfaction Problem (CSP) in the FlatZinc format
	(see https://www.minizinc.org/doc-latest/en/fzn-spec.html)
	Any FlatZinc-supporting CSP solver can then be called to get an optimal solution.
*/

package optimizer

import (
	"math"
	"os"
	"os/exec"
	"strings"

	"emperror.dev/errors"

	"github.com/rs/zerolog"

	"fybrik.io/fybrik/pkg/datapath"
	"fybrik.io/fybrik/pkg/environment"
)

const (
	MaxDataPathDepth = 4
)

type Optimizer struct {
	dpc         *DataPathCSP
	problemData []datapath.DataInfo
	env         *datapath.Environment
	solverPath  string
	log         *zerolog.Logger
}

func NewOptimizer(env *datapath.Environment, problemData []datapath.DataInfo, solverPath string, log *zerolog.Logger) *Optimizer {
	opt := Optimizer{dpc: NewDataPathCSP(problemData, env), problemData: problemData,
		env: env, solverPath: solverPath, log: log}
	return &opt
}

// Returns the solver's solution to the CSP problem (as a string containing assignments to all vars)
func (opt *Optimizer) getSolution(pathLength int) (string, error) {
	opt.log.Debug().Msgf("finding solution of length %d", pathLength)
	modelFile, err := opt.dpc.BuildFzModel(pathLength)
	if len(modelFile) > 0 {
		defer os.Remove(modelFile)
	}
	if err != nil {
		return "", errors.Wrap(err, "error building a model")
	}

	solverArgs := []string{modelFile}
	additionalArgs := environment.GetCSPArgs()
	if additionalArgs != "" {
		solverArgs = append(solverArgs, strings.Split(additionalArgs, " ")...)
	}
	opt.log.Debug().Msgf("Executing %s %v", opt.solverPath, solverArgs)
	// #nosec G204 -- Avoid "Subprocess launched with variable" error
	solverSolution, err := exec.Command(opt.solverPath, solverArgs...).Output()
	if err != nil {
		return "", errors.Wrapf(err, "error executing %s %v", opt.solverPath, solverArgs)
	}
	return string(solverSolution), nil
}

// The main method to call for finding a legal and optimal data plane
// Attempts short data-paths first, and gradually increases data-path length.
// Returns a slice of data-paths, one for each dataset (modules in different paths may overlap)
func (opt *Optimizer) Solve() ([]datapath.Solution, error) {
	bestScore := math.NaN()
	bestSolution := []datapath.Solution{}
	for pathLen := 1; pathLen <= MaxDataPathDepth; pathLen++ {
		solverSolution, err := opt.getSolution(pathLen)
		if err != nil {
			return nil, err
		}
		solution, score, err := opt.dpc.decodeSolverSolution(solverSolution, pathLen)
		if err != nil {
			return nil, err
		}
		if len(solution) > 0 && math.IsNaN(score) { // no optimization goal is specified. prefer shorter paths
			return solution, nil
		}
		if !math.IsNaN(score) && (math.IsNaN(bestScore) || score < bestScore) {
			bestScore = score
			bestSolution = solution
		}
	}
	return bestSolution, nil
}
