package controller

import (
	"fmt"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

type runtimeProbePaths struct {
	Startup   string
	Readiness string
	Liveness  string
}

type runtimeAdapter interface {
	RuntimeType() servingv1alpha1.RuntimeType

	ProbePaths() runtimeProbePaths

	BuildArgs(
		modelService *servingv1alpha1.ModelService,
	) []string
}

type fixedRuntimeAdapter struct {
	runtimeType servingv1alpha1.RuntimeType
	probePaths  runtimeProbePaths
}

func (a fixedRuntimeAdapter) RuntimeType() servingv1alpha1.RuntimeType {
	return a.runtimeType
}

func (a fixedRuntimeAdapter) ProbePaths() runtimeProbePaths {
	return a.probePaths
}

func (a fixedRuntimeAdapter) BuildArgs(
	modelService *servingv1alpha1.ModelService,
) []string {
	return append(
		[]string(nil),
		modelService.Spec.Runtime.Args...,
	)
}

func resolveRuntimeAdapter(
	runtimeType servingv1alpha1.RuntimeType,
) (runtimeAdapter, error) {
	switch string(runtimeType) {
	case "transformers":
		return fixedRuntimeAdapter{
			runtimeType: runtimeType,
			probePaths: runtimeProbePaths{
				Startup:   "/health",
				Readiness: "/ready",
				Liveness:  "/health",
			},
		}, nil

	case "kvcache-serve":
		return fixedRuntimeAdapter{
			runtimeType: runtimeType,
			probePaths: runtimeProbePaths{
				Startup:   "/health",
				Readiness: "/ready",
				Liveness:  "/health",
			},
		}, nil

	case "vllm":
		return fixedRuntimeAdapter{
			runtimeType: runtimeType,
			probePaths: runtimeProbePaths{
				Startup:   "/health",
				Readiness: "/health",
				Liveness:  "/health",
			},
		}, nil

	default:
		return nil, fmt.Errorf(
			"unsupported runtime type %q",
			runtimeType,
		)
	}
}
