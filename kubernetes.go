package zenoss

import "fmt"

const (
	// K8sClusterField is an optional metadata field containing a Kubernetes cluster.
	K8sClusterField = "k8s.cluster"

	// K8sNamespaceField is an optional metadata field containing a Kubernetes namespace.
	K8sNamespaceField = "k8s.namespace"

	// K8sPodField is an optional metadata field containing a Kubernetes pod.
	K8sPodField = "k8s.pod"

	// ImpactFromDimensionsField is an optional model metadata field.
	ImpactFromDimensionsField = "impactFromDimensions"

	// ImpactToDimensionsField is an optional model metadata field.
	ImpactToDimensionsField = "impactToDimensions"
)

func addKubernetesImpacts(metadataFields map[string]string) {
	cluster, exists := metadataFields[K8sClusterField]
	if !exists || cluster == "" {
		return
	}

	namespace, exists := metadataFields[K8sNamespaceField]
	if !exists || namespace == "" {
		return
	}

	pod, exists := metadataFields[K8sPodField]
	if !exists || pod == "" {
		return
	}

	metadataFields[ImpactFromDimensionsField] = fmt.Sprintf(
		"%s=%s,%s=%s,%s=%s",
		K8sClusterField, cluster,
		K8sNamespaceField, namespace,
		K8sPodField, pod)
}
