package controller

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servingv1alpha1 "github.com/wuge-xu/modelfleet/api/v1alpha1"
)

func TestResolveRuntimeAdapter(t *testing.T) {
	tests := []struct {
		name              string
		runtimeType       servingv1alpha1.RuntimeType
		expectedStartup   string
		expectedReadiness string
		expectedLiveness  string
		assertType        func(t *testing.T, adapter runtimeAdapter)
	}{
		{
			name:              "transformers",
			runtimeType:       servingv1alpha1.RuntimeTypeTransformers,
			expectedStartup:   "/health",
			expectedReadiness: "/ready",
			expectedLiveness:  "/health",
			assertType: func(
				t *testing.T,
				adapter runtimeAdapter,
			) {
				t.Helper()

				if _, ok := adapter.(transformersRuntimeAdapter); !ok {
					t.Fatalf(
						"expected transformersRuntimeAdapter, got %T",
						adapter,
					)
				}
			},
		},
		{
			name: "kvcache serve",
			runtimeType: servingv1alpha1.RuntimeType(
				"kvcache-serve",
			),
			expectedStartup:   "/health",
			expectedReadiness: "/ready",
			expectedLiveness:  "/health",
			assertType: func(
				t *testing.T,
				adapter runtimeAdapter,
			) {
				t.Helper()

				if _, ok := adapter.(kvCacheServeRuntimeAdapter); !ok {
					t.Fatalf(
						"expected kvCacheServeRuntimeAdapter, got %T",
						adapter,
					)
				}
			},
		},
		{
			name:              "vllm",
			runtimeType:       servingv1alpha1.RuntimeTypeVLLM,
			expectedStartup:   "/health",
			expectedReadiness: "/health",
			expectedLiveness:  "/health",
			assertType: func(
				t *testing.T,
				adapter runtimeAdapter,
			) {
				t.Helper()

				if _, ok := adapter.(vLLMRuntimeAdapter); !ok {
					t.Fatalf(
						"expected vLLMRuntimeAdapter, got %T",
						adapter,
					)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter, err := resolveRuntimeAdapter(
				test.runtimeType,
			)
			if err != nil {
				t.Fatalf(
					"resolve runtime adapter: %v",
					err,
				)
			}

			test.assertType(t, adapter)

			if adapter.RuntimeType() != test.runtimeType {
				t.Fatalf(
					"expected runtime type %q, got %q",
					test.runtimeType,
					adapter.RuntimeType(),
				)
			}

			probePaths := adapter.ProbePaths()

			if probePaths.Startup != test.expectedStartup {
				t.Fatalf(
					"expected startup path %q, got %q",
					test.expectedStartup,
					probePaths.Startup,
				)
			}

			if probePaths.Readiness != test.expectedReadiness {
				t.Fatalf(
					"expected readiness path %q, got %q",
					test.expectedReadiness,
					probePaths.Readiness,
				)
			}

			if probePaths.Liveness != test.expectedLiveness {
				t.Fatalf(
					"expected liveness path %q, got %q",
					test.expectedLiveness,
					probePaths.Liveness,
				)
			}
		})
	}
}

func TestSimpleRuntimeAdaptersBuildArgsReturnCopy(
	t *testing.T,
) {
	runtimeTypes := []servingv1alpha1.RuntimeType{
		servingv1alpha1.RuntimeTypeTransformers,
		servingv1alpha1.RuntimeType("kvcache-serve"),
	}

	for _, runtimeType := range runtimeTypes {
		t.Run(string(runtimeType), func(t *testing.T) {
			adapter, err := resolveRuntimeAdapter(
				runtimeType,
			)
			if err != nil {
				t.Fatalf(
					"resolve runtime adapter: %v",
					err,
				)
			}

			modelService := &servingv1alpha1.ModelService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "adapter-model",
					Namespace: "default",
				},
				Spec: servingv1alpha1.ModelServiceSpec{
					Runtime: servingv1alpha1.RuntimeSpec{
						Type: runtimeType,
						Args: []string{
							"--example=value",
						},
					},
				},
			}

			args := adapter.BuildArgs(modelService)

			if len(args) != 1 {
				t.Fatalf(
					"expected one argument, got %d",
					len(args),
				)
			}

			if args[0] != "--example=value" {
				t.Fatalf(
					"unexpected argument %q",
					args[0],
				)
			}

			args[0] = "--changed"

			if modelService.Spec.Runtime.Args[0] !=
				"--example=value" {
				t.Fatal(
					"adapter must return a copy of runtime arguments",
				)
			}
		})
	}
}

func TestVLLMRuntimeAdapterBuildArgsFromHFURI(
	t *testing.T,
) {
	adapter := vLLMRuntimeAdapter{}

	modelService := &servingv1alpha1.ModelService{
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				Name: "qwen",
				URI:  "hf://Qwen/Qwen3-0.6B",
			},
			Runtime: servingv1alpha1.RuntimeSpec{
				Type: servingv1alpha1.RuntimeTypeVLLM,
				Args: []string{
					"--tensor-parallel-size=1",
				},
			},
			Port: 9000,
		},
	}

	args := adapter.BuildArgs(modelService)

	expected := []string{
		"--model",
		"Qwen/Qwen3-0.6B",
		"--port",
		"9000",
		"--tensor-parallel-size=1",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Fatalf(
			"expected args %v, got %v",
			expected,
			args,
		)
	}
}

func TestVLLMModelReferencePreservesLocalPath(
	t *testing.T,
) {
	modelService := &servingv1alpha1.ModelService{
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				Name: "local-model",
				URI:  "/models/qwen",
			},
		},
	}

	modelReference := vLLMModelReference(
		modelService,
	)

	if modelReference != "/models/qwen" {
		t.Fatalf(
			"expected local model path, got %q",
			modelReference,
		)
	}
}

func TestVLLMModelReferenceFallsBackToModelName(
	t *testing.T,
) {
	modelService := &servingv1alpha1.ModelService{
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				Name: "Qwen/Qwen3-0.6B",
			},
		},
	}

	modelReference := vLLMModelReference(
		modelService,
	)

	if modelReference != "Qwen/Qwen3-0.6B" {
		t.Fatalf(
			"expected model name fallback, got %q",
			modelReference,
		)
	}
}

func TestVLLMRuntimeAdapterUsesDefaultPort(
	t *testing.T,
) {
	adapter := vLLMRuntimeAdapter{}

	modelService := &servingv1alpha1.ModelService{
		Spec: servingv1alpha1.ModelServiceSpec{
			Model: servingv1alpha1.ModelSpec{
				URI: "hf://Qwen/Qwen3-0.6B",
			},
		},
	}

	args := adapter.BuildArgs(modelService)

	expected := []string{
		"--model",
		"Qwen/Qwen3-0.6B",
		"--port",
		"8000",
	}

	if !reflect.DeepEqual(args, expected) {
		t.Fatalf(
			"expected args %v, got %v",
			expected,
			args,
		)
	}
}

func TestResolveRuntimeAdapterRejectsUnsupportedRuntime(
	t *testing.T,
) {
	_, err := resolveRuntimeAdapter(
		servingv1alpha1.RuntimeType("unsupported"),
	)

	if err == nil {
		t.Fatal(
			"expected unsupported runtime error",
		)
	}
}
