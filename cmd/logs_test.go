package cmd

import (
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGetPods(t *testing.T) {
	testCases := []struct {
		name               string
		pods               []runtime.Object
		targetNamespace    string
		targetPod          string
		targetLabelKey     string
		expectedLabelValue string
		expectSuccess      bool
	}{
		{
			name: "existing_pod_found",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "pod1-existing",
						Namespace: "namespace1",
						Labels: map[string]string{
							"label1": "value1",
						},
					},
				},
			},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      true,
		},
		{
			name:               "no_pods_existing",
			pods:               []runtime.Object{},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      false,
		},
		{
			name: "existing_pod_missing_label",
			pods: []runtime.Object{
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "existing-no-label",
						Namespace: "namespace1",
					},
				},
			},
			targetNamespace:    "namespace1",
			targetPod:          "pod1",
			targetLabelKey:     "label1",
			expectedLabelValue: "VALUE1",
			expectSuccess:      true,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeClientset := fake.NewSimpleClientset(test.pods...)
			pods, err := getPods(
				test.targetNamespace,
				fakeClientset,
			)

			for _, pod := range pods {
				fmt.Println(pod.ObjectMeta.Name)
				if pod.ObjectMeta.Name == "" && test.expectSuccess {
					t.Fatalf("no pod found: %v", err)
				}
			}
		})
	}
}

/*
				if err != nil && test.expectSuccess {
					t.Fatalf("unexpected error getting label: %v", err)
				}

						} else if err == nil && !test.expectSuccess {
							t.Fatalf("expected error but received none getting label")
						} else if labelValue != test.expectedLabelValue && test.expectSuccess {
							t.Fatalf("label value %s unexpectedly not equal to %s", labelValue, test.expectedLabelValue)
						} else if labelValue == test.expectedLabelValue && !test.expectSuccess {
							t.Fatalf("label values are unexpectedly equal: %s", labelValue)
						}
		})
	}
}
*/

//test.targetPod,
//test.targetLabelKey,
