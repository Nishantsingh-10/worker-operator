package deploy

import (
	"context"

	"github.com/kubeslice/worker-operator/controllers"
	"github.com/kubeslice/worker-operator/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type webhookClientDeploy struct {
}

func NewWebhookClient() *webhookClientDeploy {
	return &webhookClientDeploy{}
}

func (w *webhookClientDeploy) SliceAppNamespaceConfigured(ctx context.Context, slice string, namespace string) (bool, error) {
	return controllers.SliceAppNamespaceConfigured(ctx, slice, namespace)
}

func (w *webhookClientDeploy) GetNamespaceLabels(ctx context.Context, client client.Client, namespace string) (map[string]string, error) {
	log := logger.NewLogger().WithName("webhook logger")
	nS := &corev1.Namespace{}
	err := client.Get(context.Background(), types.NamespacedName{Name: namespace}, nS)
	if err != nil {
		log.Info("Failed to find namespace", "namespace", namespace)
		return nil, err
	}
	nsLabels := nS.ObjectMeta.GetLabels()
	return nsLabels, nil
}
