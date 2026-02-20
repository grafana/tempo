package process

import (
	"sort"

	"github.com/grafana/tanka/pkg/kubernetes/manifest"
)

// Order in which install different kinds of Kubernetes objects.
// Inspired by https://github.com/helm/helm/blob/8c84a0bc0376650bc3d7334eef0c46356c22fa36/pkg/releaseutil/kind_sorter.go
var kindOrder = []string{
	"Namespace",
	"NetworkPolicy",
	"ResourceQuota",
	"LimitRange",
	"PodSecurityPolicy",
	"PodDisruptionBudget",
	"ServiceAccount",
	"Secret",
	"ConfigMap",
	"StorageClass",
	"PersistentVolume",
	"PersistentVolumeClaim",
	"CustomResourceDefinition",
	"ClusterRole",
	"ClusterRoleList",
	"ClusterRoleBinding",
	"ClusterRoleBindingList",
	"Role",
	"RoleList",
	"RoleBinding",
	"RoleBindingList",
	"Service",
	"DaemonSet",
	"Pod",
	"ReplicationController",
	"ReplicaSet",
	"Deployment",
	"HorizontalPodAutoscaler",
	"StatefulSet",
	"Job",
	"CronJob",
	"Ingress",
	"APIService",
}

// Sort orders manifests in a stable order, taking order-dependencies of these
// into consideration. This is best-effort based:
// - Use the static kindOrder list if possible
// - Sort alphabetically by kind otherwise
// - If kind equal, sort alphabetically by name
func Sort(list manifest.List) {
	sort.SliceStable(list, func(i int, j int) bool {
		var io, jo int

		// anything that is not in kindOrder will get to the end of the install list.
		for io = 0; io < len(kindOrder); io++ {
			if list[i].Kind() == kindOrder[io] {
				break
			}
		}

		for jo = 0; jo < len(kindOrder); jo++ {
			if list[j].Kind() == kindOrder[jo] {
				break
			}
		}

		// If Kind of both objects are at different indexes of kindOrder, sort by them
		if io != jo {
			return io < jo
		}

		// If the Kinds themselves are different (e.g. both of the Kinds are not in
		// the kindOrder), order alphabetically.
		if list[i].Kind() != list[j].Kind() {
			return list[i].Kind() < list[j].Kind()
		}

		// If namespaces differ, sort by the namespace.
		if list[i].Metadata().Namespace() != list[j].Metadata().Namespace() {
			return list[i].Metadata().Namespace() < list[j].Metadata().Namespace()
		}

		if list[i].Metadata().Name() != list[j].Metadata().Name() {
			return list[i].Metadata().Name() < list[j].Metadata().Name()
		}

		return list[i].Metadata().GenerateName() < list[j].Metadata().GenerateName()
	})
}
