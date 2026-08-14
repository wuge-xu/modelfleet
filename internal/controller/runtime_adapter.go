package controller

import (
	"fmt"
	"strconv"
	"strings"

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

type transformersRuntimeAdapter struct{}

type kvCacheServeRuntimeAdapter struct{}

type vLLMRuntimeAdapter struct{}

var _ runtimeAdapter = transformersRuntimeAdapter{}
var _ runtimeAdapter = kvCacheServeRuntimeAdapter{}
var _ runtimeAdapter = vLLMRuntimeAdapter{}

func (transformersRuntimeAdapter) RuntimeType() servingv1alpha1.RuntimeType {
	return servingv1alpha1.RuntimeTypeTransformers
}

func (transformersRuntimeAdapter) ProbePaths() runtimeProbePaths {
	return runtimeProbePaths{
		Startup:   "/health",
		Readiness: "/ready",
		Liveness:  "/health",
	}
}

func (transformersRuntimeAdapter) BuildArgs(
	modelService *servingv1alpha1.ModelService,
) []string {
	return copyRuntimeArgs(modelService)
}

func (kvCacheServeRuntimeAdapter) RuntimeType() servingv1alpha1.RuntimeType {
	return servingv1alpha1.RuntimeType(
		"kvcache-serve",
	)
}

func (kvCacheServeRuntimeAdapter) ProbePaths() runtimeProbePaths {
	return runtimeProbePaths{
		Startup:   "/health",
		Readiness: "/ready",
		Liveness:  "/health",
	}
}

func (kvCacheServeRuntimeAdapter) BuildArgs(
	modelService *servingv1alpha1.ModelService,
) []string {
	return copyRuntimeArgs(modelService)
}

func (vLLMRuntimeAdapter) RuntimeType() servingv1alpha1.RuntimeType {
	return servingv1alpha1.RuntimeTypeVLLM
}

func (vLLMRuntimeAdapter) ProbePaths() runtimeProbePaths {
	return runtimeProbePaths{
		Startup:   "/health",
		Readiness: "/health",
		Liveness:  "/health",
	}
}

func (vLLMRuntimeAdapter) BuildArgs(
	modelService *servingv1alpha1.ModelService,
) []string {
	args := []string{
		"--model",
		vLLMModelReference(modelService),
		"--port",
		strconv.FormatInt(
			int64(desiredModelPort(modelService)),
			10,
		),
	}

	return append(
		args,
		copyRuntimeArgs(modelService)...,
	)
}

func resolveRuntimeAdapter(
	runtimeType servingv1alpha1.RuntimeType,
) (runtimeAdapter, error) {
	switch runtimeType {
	case servingv1alpha1.RuntimeTypeTransformers:
		return transformersRuntimeAdapter{}, nil

	case servingv1alpha1.RuntimeType("kvcache-serve"):
		return kvCacheServeRuntimeAdapter{}, nil

	case servingv1alpha1.RuntimeTypeVLLM:
		return vLLMRuntimeAdapter{}, nil

	default:
		return nil, fmt.Errorf(
			"unsupported runtime type %q",
			runtimeType,
		)
	}
}

func vLLMModelReference(
	modelService *servingv1alpha1.ModelService,
) string {
	modelURI := strings.TrimSpace(
		modelService.Spec.Model.URI,
	)

	if strings.HasPrefix(modelURI, "hf://") {
		return strings.TrimPrefix(
			modelURI,
			"hf://",
		)
	}

	if modelURI != "" {
		return modelURI
	}

	return modelService.Spec.Model.Name
}

func copyRuntimeArgs(
	modelService *servingv1alpha1.ModelService,
) []string {
	return append(
		[]string(nil),
		modelService.Spec.Runtime.Args...,
	)
}
