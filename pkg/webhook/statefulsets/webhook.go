package statefulsets

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/kubeslice/worker-operator/controllers"
	"github.com/kubeslice/worker-operator/pkg/logger"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// var (
// 	log = logger.NewLogger().WithName("deployment Webhook").V(1)
// )

const (
	admissionWebhookAnnotationInjectKey       = controllers.ApplicationNamespaceSelectorLabelKey
	AdmissionWebhookAnnotationStatusKey       = "kubeslice.io/status"
	PodInjectLabelKey                         = "kubeslice.io/pod-type"
	admissionWebhookSliceNamespaceSelectorKey = controllers.ApplicationNamespaceSelectorLabelKey
	controlPlaneNamespace                     = "kubeslice-system"
	nsmInjectAnnotaionKey                     = "ns.networkservicemesh.io"
)

type SliceInfoProvider interface {
	SliceAppNamespaceConfigured(ctx context.Context, slice string, namespace string) (bool, error)
	GetNamespaceLabels(ctx context.Context, client client.Client, namespace string) (map[string]string, error)
}

type WebhookServer struct {
	Client          client.Client
	decoder         *admission.Decoder
	SliceInfoClient SliceInfoProvider
}

func (wh *WebhookServer) Handle(ctx context.Context, req admission.Request) admission.Response {
	statefulset := &appsv1.StatefulSet{}
	err := wh.decoder.Decode(req, statefulset)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	log := logger.FromContext(ctx)

	if mutate, sliceName := wh.MutationRequired(statefulset.ObjectMeta, ctx); !mutate {
		log.Info("mutation not required", "pod metadata", statefulset.Spec.Template.ObjectMeta)
	} else {
		log.Info("mutating deploy", "pod metadata", statefulset.Spec.Template.ObjectMeta)
		statefulset = Mutate(statefulset, sliceName)
		log.Info("mutated deploy", "pod metadata", statefulset.Spec.Template.ObjectMeta)
	}

	marshaled, err := json.Marshal(statefulset)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func (wh *WebhookServer) InjectDecoder(d *admission.Decoder) error {
	wh.decoder = d
	return nil
}

func Mutate(statefulset *appsv1.StatefulSet, sliceName string) *appsv1.StatefulSet {
	// Add injection status to deployment annotations
	statefulset.Annotations[AdmissionWebhookAnnotationStatusKey] = "injected"

	if statefulset.Spec.Template.ObjectMeta.Annotations == nil {
		statefulset.Spec.Template.ObjectMeta.Annotations = map[string]string{}
	}

	// Add vl3 annotation to pod template
	annotations := statefulset.Spec.Template.ObjectMeta.Annotations
	annotations[nsmInjectAnnotaionKey] = "vl3-service-" + sliceName

	// Add slice identifier labels to pod template
	labels := statefulset.Spec.Template.ObjectMeta.Labels
	labels[PodInjectLabelKey] = "app"
	labels[admissionWebhookAnnotationInjectKey] = sliceName

	return statefulset
}

func (wh *WebhookServer) MutationRequired(metadata metav1.ObjectMeta, ctx context.Context) (bool, string) {
	log := logger.FromContext(ctx)
	annotations := metadata.GetAnnotations()
	//early exit if metadata in nil
	//we allow empty annotation, but namespace should not be empty
	if metadata.GetNamespace() == "" {
		log.Info("namespace is empty")
		return false, ""
	}
	// do not inject if it is already injected
	if annotations[AdmissionWebhookAnnotationStatusKey] == "injected" {
		log.Info("Deployment is already injected")
		return false, ""
	}

	// Do not auto onboard control plane namespace. Ideally, we should not have any deployment/pod in the control plane ns connect to a slice
	if metadata.Namespace == controlPlaneNamespace {
		log.Info("namespace is same as controle plane namespace")
		return false, ""
	}

	nsLabels, err := wh.SliceInfoClient.GetNamespaceLabels(context.Background(), wh.Client, metadata.Namespace)
	if err != nil {
		log.Error(err, "Error getting namespace labels")
		return false, ""
	}
	if nsLabels == nil {
		log.Info("Namespace has no labels")
		return false, ""
	}

	sliceNameInNs, sliceLabelPresent := nsLabels[admissionWebhookSliceNamespaceSelectorKey]
	if !sliceLabelPresent {
		log.Info("Namespace has no slice labels")
		return false, ""
	}

	nsConfigured, err := wh.SliceInfoClient.SliceAppNamespaceConfigured(context.Background(), sliceNameInNs, metadata.Namespace)
	if err != nil {
		log.Error(err, "Failed to get app namespace info for slice",
			"slice", sliceNameInNs, "namespace", metadata.Namespace)
		return false, ""
	}
	if !nsConfigured {
		log.Info("Namespace not part of slice", "namespace", metadata.Namespace, "slice", sliceNameInNs)
		return false, ""
	}
	// The annotation kubeslice.io/slice:SLICENAME is present, enable mutation
	return true, sliceNameInNs
}
