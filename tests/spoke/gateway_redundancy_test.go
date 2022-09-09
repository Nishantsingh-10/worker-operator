/*
 *  Copyright (c) 2022 Avesha, Inc. All rights reserved.
 *
 *  SPDX-License-Identifier: Apache-2.0
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package spoke_test

import (
	"context"
	"time"

	hubv1alpha1 "github.com/kubeslice/apis/pkg/controller/v1alpha1"
	kubeslicev1beta1 "github.com/kubeslice/worker-operator/api/v1beta1"
	nsmv1alpha1 "github.com/networkservicemesh/networkservicemesh/k8s/pkg/apis/networkservice/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "sigs.k8s.io/controller-runtime/pkg/client"
)

// var sliceGwFinalizer = []string{
// 	"networking.kubeslice.io/slicegw-finalizer"}

var _ = Describe("Worker SlicegwController", func() {

	var sliceGw *kubeslicev1beta1.SliceGateway
	var createdSliceGw *kubeslicev1beta1.SliceGateway
	var slice *kubeslicev1beta1.Slice
	var svc *corev1.Service
	var createdSlice *kubeslicev1beta1.Slice
	var vl3ServiceEndpoint *nsmv1alpha1.NetworkServiceEndpoint
	var appPod *corev1.Pod
	var ns *corev1.Namespace
	var cluster *hubv1alpha1.Cluster
	var nsmconfig *corev1.ConfigMap
	Context("With SliceGW CR created", func() {

		BeforeEach(func() {

			sliceGw = &kubeslicev1beta1.SliceGateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-slicegw",
					Namespace: CONTROL_PLANE_NS,
				},
				Spec: kubeslicev1beta1.SliceGatewaySpec{
					SliceName: "test-slice-4",
				},
			}
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: PROJECT_NS,
				},
				Spec: corev1.NamespaceSpec{},
			}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ns.Name}, ns)
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, ns)).Should(Succeed())
			}
			cluster = &hubv1alpha1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-internal-node",
					Namespace: PROJECT_NS,
				},
				Spec:   hubv1alpha1.ClusterSpec{},
				Status: hubv1alpha1.ClusterStatus{},
			}
			Expect(k8sClient.Create(ctx, cluster)).Should(Succeed())
			slice = &kubeslicev1beta1.Slice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-slice-4",
					Namespace: CONTROL_PLANE_NS,
				},
				Spec: kubeslicev1beta1.SliceSpec{},
			}
			svc = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubeslice-dns",
					Namespace: "kubeslice-system",
				},
				Spec: corev1.ServiceSpec{
					ClusterIP: "10.0.0.20",
					Ports: []corev1.ServicePort{{
						Port: 52,
					}},
				},
			}
			labels := map[string]string{
				"kubeslice.io/slice":         "test-slice-4",
				"kubeslice.io/pod-type":      "slicegateway",
				"networkservicemesh.io/app":  "test-slicegw",
				"networkservicemesh.io/impl": "vl3-service-test-slice-4",
			}

			ann := map[string]string{
				"ns.networkservicemesh.io": "vl3-service-test-slice",
			}
			appPod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "nginx-pod",
					Namespace:   CONTROL_PLANE_NS,
					Labels:      labels,
					Annotations: ann,
				},
				Spec: corev1.PodSpec{

					Containers: []corev1.Container{
						{
							Image: "nginx",
							Name:  "nginx",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			}

			vl3ServiceEndpoint = &nsmv1alpha1.NetworkServiceEndpoint{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "networkservicemesh.io/v1alpha1",
					Kind:       "NetworkServiceEndpoint",
				},
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "vl3-service-" + "test-slice-4",
					Namespace:    "kubeslice-system",
					Labels: map[string]string{
						"app":                "vl3-nse-" + "test-slice-4",
						"networkservicename": "vl3-service-" + "test-slice-4",
					},
				},
				Spec: nsmv1alpha1.NetworkServiceEndpointSpec{
					NetworkServiceName: "vl3-service-" + "test-slice-4",
					Payload:            "IP",
					NsmName:            "test-node",
				},
			}
			nsmconfig = configMap("nsm-config", "kubeslice-system", `
									prefixes:
									- 192.168.0.0/16
									- 10.96.0.0/12`)
			err = k8sClient.Get(ctx, types.NamespacedName{Name: nsmconfig.Name, Namespace: nsmconfig.Namespace}, nsmconfig)
			if errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, nsmconfig)).Should(Succeed())
			}
			createdSlice = &kubeslicev1beta1.Slice{}
			createdSliceGw = &kubeslicev1beta1.SliceGateway{}
			founddepl := &appsv1.Deployment{}
			deplKey := types.NamespacedName{Name: "test-slicegw", Namespace: CONTROL_PLANE_NS}

			DeferCleanup(func() {
				ctx := context.Background()
				Eventually(func() bool {
					err := k8sClient.Delete(ctx, cluster)
					return errors.IsNotFound(err)
				}, time.Second*10, time.Millisecond*250).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, slice)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: slice.Name, Namespace: slice.Namespace}, createdSlice)
					return errors.IsNotFound(err)
				}, time.Second*30, time.Millisecond*250).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, sliceGw)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: sliceGw.Name, Namespace: sliceGw.Namespace}, createdSliceGw)
					return errors.IsNotFound(err)
				}, time.Second*30, time.Millisecond*250).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, vl3ServiceEndpoint)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: sliceGw.Name, Namespace: sliceGw.Namespace}, appPod)
					return errors.IsNotFound(err)
				}, time.Second*40, time.Millisecond*250).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, appPod)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, deplKey, founddepl)
					if err != nil {
						return errors.IsNotFound(err)
					}
					Expect(k8sClient.Delete(ctx, founddepl)).Should(Succeed())
					return true
				}, time.Second*40, time.Millisecond*250).Should(BeTrue())
				Expect(k8sClient.Delete(ctx, svc)).Should(Succeed())

			})
		})
		It("should create 2 gateway server pods ", func() {
			ctx := context.Background()

			Expect(k8sClient.Create(ctx, svc)).Should(Succeed())
			Expect(k8sClient.Create(ctx, slice)).Should(Succeed())
			Expect(k8sClient.Create(ctx, vl3ServiceEndpoint)).Should(Succeed())
			Expect(k8sClient.Create(ctx, sliceGw)).Should(Succeed())
			Expect(k8sClient.Create(ctx, appPod)).Should(Succeed())
			sliceKey := types.NamespacedName{Name: "test-slice-4", Namespace: CONTROL_PLANE_NS}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, sliceKey, createdSlice)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			slicegwkey := types.NamespacedName{Name: "test-slicegw", Namespace: CONTROL_PLANE_NS}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, slicegwkey, createdSliceGw)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			createdSliceGw.Status.Config.SliceGatewayHostType = "Server"
			// createdSliceGw.Status.
			Eventually(func() bool {
				err := k8sClient.Status().Update(ctx, createdSliceGw)
				return err == nil
			}, time.Second*30, time.Millisecond*250).Should(BeTrue())

			foundsvc := &corev1.Service{}
			svckey := types.NamespacedName{Name: "svc-test-slicegw", Namespace: CONTROL_PLANE_NS}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, svckey, foundsvc)
				return err == nil
			}, time.Second*30, time.Millisecond*250).Should(BeTrue())
			founddepl := &appsv1.Deployment{}
			deplKey := types.NamespacedName{Name: createdSliceGw.Name, Namespace: sliceGw.Namespace}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deplKey, founddepl)
				if err != nil {
					return false
				}
				return *founddepl.Spec.Replicas == 2
			}, time.Second*120, time.Millisecond*250).Should(BeTrue())
		})
		It("should create 2 gateway client pods", func() {
			ctx := context.Background()
			Expect(k8sClient.Create(ctx, svc)).Should(Succeed())
			Expect(k8sClient.Create(ctx, slice)).Should(Succeed())
			Expect(k8sClient.Create(ctx, vl3ServiceEndpoint)).Should(Succeed())
			Expect(k8sClient.Create(ctx, sliceGw)).Should(Succeed())
			Expect(k8sClient.Create(ctx, appPod)).Should(Succeed())

			sliceKey := types.NamespacedName{Name: "test-slice-4", Namespace: CONTROL_PLANE_NS}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, sliceKey, createdSlice)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			slicegwkey := types.NamespacedName{Name: "test-slicegw", Namespace: CONTROL_PLANE_NS}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, slicegwkey, createdSliceGw)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			createdSliceGw.Status.Config.SliceGatewayHostType = "Client"
			createdSliceGw.Status.Config.SliceGatewayRemoteGatewayID = "remote-gateway-id"
			createdSliceGw.Status.Config.SliceGatewayRemoteNodeIPs = []string{"192.168.1.1"}
			createdSliceGw.Status.Config.SliceGatewayRemoteNodePort = 8080

			Eventually(func() bool {
				err := k8sClient.Status().Update(ctx, createdSliceGw)
				return err == nil
			}, time.Second*30, time.Millisecond*250).Should(BeTrue())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, slicegwkey, createdSliceGw)
				return err == nil
			}, time.Second*10, time.Millisecond*250).Should(BeTrue())

			founddepl := &appsv1.Deployment{}
			deplKey := types.NamespacedName{Name: "test-slicegw", Namespace: CONTROL_PLANE_NS}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, deplKey, founddepl)
				if err != nil {
					return false
				}
				return *founddepl.Spec.Replicas == 2

			}, time.Second*40, time.Millisecond*250).Should(BeTrue())

			Expect(founddepl.Spec.Template.Spec.Containers[1].Name).Should(Equal("kubeslice-openvpn-client"))
		})
	})
})
