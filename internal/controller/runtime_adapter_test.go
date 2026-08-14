package controller

import (
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

func TestRuntimeAdaptersBuildArgsReturnCopy(t *testing.T) {
	runtimeTypes := []servingv1alpha1.RuntimeType{
		servingv1alpha1.RuntimeTypeTransformers,
		servingv1alpha1.RuntimeType("kvcache-serve"),
		servingv1alpha1.RuntimeTypeVLLM,
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
