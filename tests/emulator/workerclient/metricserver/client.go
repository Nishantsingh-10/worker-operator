package workerclient

import (
	"github.com/stretchr/testify/mock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
)

type ClientEmulator struct {
	mock.Mock
}

func NewClientEmulator() (*ClientEmulator, error) {
	return new(ClientEmulator), nil
}

func (MetricSeverClientEmulator *ClientEmulator) GetNamespaceMetrics(namespace string) (*v1beta1.PodMetricsList, error) {
	allPodMetics := v1beta1.PodMetricsList{}
	allPodMetics.Items = append(allPodMetics.Items, v1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-1",
		},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name: "test-container-1",
				Usage: v1.ResourceList{
					"cpu":    *resource.NewMilliQuantity(2, resource.DecimalSI),
					"memory": *resource.NewMilliQuantity(10, resource.BinarySI),
				},
			},
		},
	}, v1beta1.PodMetrics{
		ObjectMeta: metav1.ObjectMeta{
			Name: "pod-2",
		},
		Containers: []v1beta1.ContainerMetrics{
			{
				Name: "test-container-1",
				Usage: v1.ResourceList{
					"cpu":    *resource.NewMilliQuantity(1, resource.DecimalSI),
					"memory": *resource.NewMilliQuantity(20, resource.BinarySI),
				},
			},
			{
				Name: "test-container-2",
				Usage: v1.ResourceList{
					"cpu":    *resource.NewMilliQuantity(1, resource.DecimalSI),
					"memory": *resource.NewMilliQuantity(30, resource.BinarySI),
				},
			},
		},
	})
	return &allPodMetics, nil
}
