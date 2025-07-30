package controller

import (
	"io"
	"net/http"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *KubedeckReconciler) handlePodsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.PodList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""}) // List in all namespaces
	writeJsonResponse(w, list.Items, "pods", err)
}

func (r *KubedeckReconciler) handlePVRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.PersistentVolumeList{}
	err := r.Client.List(ctx, list, &client.ListOptions{}) // PVs are not namespaced
	writeJsonResponse(w, list.Items, "persistent volumes", err)
}

func (r *KubedeckReconciler) handlePVCRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.PersistentVolumeClaimList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "persistent volume claims", err)
}

func (r *KubedeckReconciler) handleConfigMapsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.ConfigMapList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "configmaps", err)
}

func (r *KubedeckReconciler) handleServicesRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.ServiceList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "services", err)
}

func (r *KubedeckReconciler) handleIngressesRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &networkingv1.IngressList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "ingresses", err)
}

func (r *KubedeckReconciler) handleDeploymentsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &appsv1.DeploymentList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "deployments", err)
}

func (r *KubedeckReconciler) handleStatefulSetsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &appsv1.StatefulSetList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "statefulsets", err)
}

func (r *KubedeckReconciler) handleDaemonSetsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &appsv1.DaemonSetList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "daemonsets", err)
}

func (r *KubedeckReconciler) handleReplicaSetsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &appsv1.ReplicaSetList{}
	err := r.Client.List(ctx, list, &client.ListOptions{Namespace: ""})
	writeJsonResponse(w, list.Items, "replicasets", err)
}

func (r *KubedeckReconciler) handleNamespacesRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.NamespaceList{}
	err := r.Client.List(ctx, list, &client.ListOptions{}) // Namespaces are not namespaced
	writeJsonResponse(w, list.Items, "namespaces", err)
}

func (r *KubedeckReconciler) handleNodesRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &corev1.NodeList{}
	err := r.Client.List(ctx, list, &client.ListOptions{}) // Nodes are not namespaced
	writeJsonResponse(w, list.Items, "nodes", err)
}

func (r *KubedeckReconciler) handleStorageClassesRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	list := &storagev1.StorageClassList{}
	err := r.Client.List(ctx, list, &client.ListOptions{}) // StorageClasses are not namespaced
	writeJsonResponse(w, list.Items, "storageclasses", err)
}

func (r *KubedeckReconciler) handlePodLogsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context() // Use request context for cancellation propagation
	log := webServerLog.WithName("handlePodLogsRequest")

	podName := req.URL.Query().Get("pod")
	namespace := req.URL.Query().Get("namespace")
	containerName := req.URL.Query().Get("container") // Optional: specify container

	if podName == "" || namespace == "" {
		log.Info("Missing 'pod' or 'namespace' query parameter")
		http.Error(w, "Query parameters 'pod' and 'namespace' are required.", http.StatusBadRequest)
		return
	}

	log.Info("Fetching logs", "pod", podName, "namespace", namespace, "container", containerName)

	// Create a standard Kubernetes clientset using the rest.Config from the reconciler
	clientset, err := kubernetes.NewForConfig(r.Config)
	if err != nil {
		log.Error(err, "Failed to create Kubernetes clientset")
		http.Error(w, "Internal server error: failed to create Kubernetes clientset", http.StatusInternalServerError)
		return
	}

	podLogOpts := corev1.PodLogOptions{}
	if containerName != "" {
		podLogOpts.Container = containerName
	}
	// Можно добавить другие опции, например:
	// podLogOpts.Follow = true // Для потоковой передачи в реальном времени (потребует другой обработки)
	// podLogOpts.TailLines = &someInt64Value // Количество последних строк
	// podLogOpts.Timestamps = true

	// Строим запрос на получение логов
	logRequest := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)

	// Получаем поток логов
	logStream, err := logRequest.Stream(ctx) // Pass context here
	if err != nil {
		log.Error(err, "Failed to stream pod logs", "pod", podName, "namespace", namespace)
		http.Error(w, "Failed to stream pod logs: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer logStream.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	// Копируем поток логов напрямую в ответ
	_, err = io.Copy(w, logStream)
	if err != nil {
		// Ошибка может произойти, если клиент отключается во время стриминга.
		// Обычно логируется, но не обязательно является фатальной ошибкой сервера.
		log.Error(err, "Error writing log stream to response", "pod", podName, "namespace", namespace)
		// Заголовки уже могли быть отправлены, поэтому отправка другого http.Error может не сработать как ожидалось.
	}
	log.Info("Successfully streamed logs", "pod", podName, "namespace", namespace)
}
