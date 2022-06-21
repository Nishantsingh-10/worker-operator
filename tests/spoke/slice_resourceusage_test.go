package spoke_test

import (
	"context"

	kubeslicev1beta1 "github.com/kubeslice/worker-operator/api/v1beta1"

	slicepkg "github.com/kubeslice/worker-operator/controllers/slice"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

var _ = Describe("SliceResourceUsage", func() {
	var slice *kubeslicev1beta1.Slice
	var createdSlice *kubeslicev1beta1.Slice
	var appNs *corev1.Namespace
	var appNs2 *corev1.Namespace

	sliceName := "test-slice-resquota"
	Context("With slice CR created and application namespaces specified ", func() {
		BeforeEach(func() {
			// Prepare k8s objects for slice
			slice = &kubeslicev1beta1.Slice{
				ObjectMeta: metav1.ObjectMeta{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				},
				Spec: kubeslicev1beta1.SliceSpec{},
			}
			appNs = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iperf1",
				},
			}
			appNs2 = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iperf2",
				},
			}

			createdSlice = &kubeslicev1beta1.Slice{}
			// Cleanup after each test
			DeferCleanup(func() {
				ctx := context.Background()
				Expect(k8sClient.Delete(ctx, slice)).Should(Succeed())
				Eventually(func() bool {
					err := k8sClient.Get(ctx, types.NamespacedName{Name: slice.Name, Namespace: slice.Namespace}, slice)
					return errors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})
		})
		It("Should update resource usage on slice config", func() {
			Expect(k8sClient.Create(ctx, slice)).Should(Succeed())
			Expect(k8sClient.Create(ctx, appNs)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				if err != nil {
					return err
				}
				createdSlice.Status.SliceConfig = &kubeslicev1beta1.SliceConfig{
					NamespaceIsolationProfile: &kubeslicev1beta1.NamespaceIsolationProfile{
						ApplicationNamespaces: []string{
							"iperf1",
						},
					},
				}
				err = k8sClient.Status().Update(ctx, createdSlice)
				return err
			})
			Expect(err).To(BeNil())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "iperf1"}, appNs)
				if err != nil {
					return false
				}
				labels := appNs.ObjectMeta.GetLabels()
				if labels == nil {
					return false
				}
				sliceLabel, ok := labels[slicepkg.ApplicationNamespaceSelectorLabelKey]
				if !ok {
					return false
				}
				return sliceLabel == slice.Name
			}, timeout, interval).Should(BeTrue())

			// check for slice resource usage
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return createdSlice.Status.SliceConfig.WorkerSliceResourceQuotaStatus != nil
			}, timeout, interval).Should(BeTrue())
		})

		It("should have more than one namespace's resource usage on slice config", func() {
			Expect(k8sClient.Create(ctx, slice)).Should(Succeed())
			Expect(k8sClient.Create(ctx, appNs2)).Should(Succeed())
			// confirm slice is created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			// label slice with iperf1 ns
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				if err != nil {
					return err
				}
				createdSlice.Status.SliceConfig = &kubeslicev1beta1.SliceConfig{
					NamespaceIsolationProfile: &kubeslicev1beta1.NamespaceIsolationProfile{
						ApplicationNamespaces: []string{
							"iperf1",
						},
					},
				}
				err = k8sClient.Status().Update(ctx, createdSlice)
				return err
			})
			Expect(err).To(BeNil())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "iperf1"}, appNs)
				if err != nil {
					return false
				}
				labels := appNs.ObjectMeta.GetLabels()
				if labels == nil {
					return false
				}
				sliceLabel, ok := labels[slicepkg.ApplicationNamespaceSelectorLabelKey]
				if !ok {
					return false
				}
				return sliceLabel == slice.Name
			}, timeout, interval).Should(BeTrue())

			// check for slice resource usage
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return createdSlice.Status.SliceConfig.WorkerSliceResourceQuotaStatus != nil
			}, timeout, interval).Should(BeTrue())

			// add iperf2 in slice config
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				if err != nil {
					return err
				}
				createdSlice.Status.SliceConfig.NamespaceIsolationProfile.ApplicationNamespaces = append(
					createdSlice.Status.SliceConfig.NamespaceIsolationProfile.ApplicationNamespaces,
					"iperf2",
				)
				err = k8sClient.Status().Update(ctx, createdSlice)
				return err
			})
			Expect(err).To(BeNil())
			//verify if the iperf2 namespace is labelled correctly
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "iperf2"}, appNs2)
				if err != nil {
					return false
				}
				labels := appNs2.ObjectMeta.GetLabels()
				if labels == nil {
					return false
				}

				sliceLabel, ok := labels[slicepkg.ApplicationNamespaceSelectorLabelKey]
				if !ok {
					return false
				}
				return sliceLabel == slice.Name
			}, timeout, interval).Should(BeTrue())

			// check for slice resource usage
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return len(createdSlice.Status.SliceConfig.WorkerSliceResourceQuotaStatus.ClusterResourceQuotaStatus.NamespaceResourceQuotaStatus) == 2
			}, timeout, interval).Should(BeTrue())

		})

		It("verify the time difference btw updates of slice config resource usage", func() {
			var configUpdateOn int64
			Expect(k8sClient.Create(ctx, slice)).Should(Succeed())
			// confirm slice is created
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			// add iperf1 ns in slice
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				if err != nil {
					return err
				}
				createdSlice.Status.SliceConfig = &kubeslicev1beta1.SliceConfig{
					NamespaceIsolationProfile: &kubeslicev1beta1.NamespaceIsolationProfile{
						ApplicationNamespaces: []string{
							"iperf1",
						},
					},
				}
				err = k8sClient.Status().Update(ctx, createdSlice)
				return err
			})
			Expect(err).To(BeNil())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "iperf1"}, appNs)
				if err != nil {
					return false
				}
				labels := appNs.ObjectMeta.GetLabels()
				if labels == nil {
					return false
				}
				sliceLabel, ok := labels[slicepkg.ApplicationNamespaceSelectorLabelKey]
				if !ok {
					return false
				}
				return sliceLabel == slice.Name
			}, timeout, interval).Should(BeTrue())

			// check for slice resource usage
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				configUpdateOn = createdSlice.Status.ConfigUpdatedOn
				return createdSlice.Status.SliceConfig.WorkerSliceResourceQuotaStatus != nil
			}, timeout, interval).Should(BeTrue())

			// add iperf2 in slice config
			err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				if err != nil {
					return err
				}
				createdSlice.Status.SliceConfig.NamespaceIsolationProfile.ApplicationNamespaces = append(
					createdSlice.Status.SliceConfig.NamespaceIsolationProfile.ApplicationNamespaces,
					"iperf2",
				)
				err = k8sClient.Status().Update(ctx, createdSlice)
				return err
			})
			Expect(err).To(BeNil())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: "iperf2"}, appNs2)
				if err != nil {
					return false
				}
				labels := appNs2.ObjectMeta.GetLabels()
				if labels == nil {
					return false
				}

				sliceLabel, ok := labels[slicepkg.ApplicationNamespaceSelectorLabelKey]
				if !ok {
					return false
				}
				return sliceLabel == slice.Name
			}, timeout, interval).Should(BeTrue())

			// check for slice resource usage
			Eventually(func() bool {
				k8sClient.Get(ctx, types.NamespacedName{
					Name:      sliceName,
					Namespace: CONTROL_PLANE_NS,
				}, createdSlice)
				return len(createdSlice.Status.SliceConfig.WorkerSliceResourceQuotaStatus.ClusterResourceQuotaStatus.NamespaceResourceQuotaStatus) > 1
			}, timeout, interval).Should(BeTrue())
			// minimum 60 seconds difference
			Expect(createdSlice.Status.ConfigUpdatedOn).To(SatisfyAny(BeNumerically(">", configUpdateOn+60), Equal(configUpdateOn+60)))

		})
	})
})
