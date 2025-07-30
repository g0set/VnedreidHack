// update_handlers.go
package controller // Используйте то же имя пакета, что и в handlers.go

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/controller-runtime/pkg/client" // Уже должно быть в APIHandlers
)

func (h *KubedeckReconciler) HandleUpdateForFront(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace string `json:"namespace"`
		Resource  struct {
			ResourceType  string `json:"resourceType"`
			Name          string `json:"name"`
			CPURequest    int    `json:"cpuRequest"`
			MemoryRequest int    `json:"memoryRequest"`
			CPULimit      int    `json:"cpuLimit"`
			MemoryLimit   int    `json:"memoryLimit"`
			Replicas      int32  `json:"replicas"`
		} `json:"resource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}
	defer r.Body.Close()

	ctx := r.Context()

	// Если ReplicaSet — обрезаем имя
	if payload.Resource.ResourceType == "ReplicaSet" {
		if i := strings.LastIndex(payload.Resource.Name, "-"); i != -1 {
			payload.Resource.Name = payload.Resource.Name[:i]
		}
	}

	// Обновляет только те поля container.Resources, которые явно указаны (не -1)
	applyResources := func(container *corev1.Container) {
		if container.Resources.Requests == nil {
			container.Resources.Requests = corev1.ResourceList{}
		}
		if container.Resources.Limits == nil {
			container.Resources.Limits = corev1.ResourceList{}
		}

		if payload.Resource.CPURequest >= 0 {
			container.Resources.Requests[corev1.ResourceCPU] =
				resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPURequest))
		}
		if payload.Resource.MemoryRequest >= 0 {
			container.Resources.Requests[corev1.ResourceMemory] =
				resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryRequest))
		}
		if payload.Resource.CPULimit >= 0 {
			container.Resources.Limits[corev1.ResourceCPU] =
				resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPULimit))
		}
		if payload.Resource.MemoryLimit >= 0 {
			container.Resources.Limits[corev1.ResourceMemory] =
				resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryLimit))
		}
	}

	// Универсальная обработка для всех 4 типов
	updateWorkload := func(obj client.Object, containers []corev1.Container, replicas *int32) {
		if len(containers) > 0 {
			applyResources(&containers[0])
		}
		if payload.Resource.Replicas >= 0 && replicas != nil {
			*replicas = payload.Resource.Replicas
		}
		if err := h.Client.Update(ctx, obj); err != nil {
			h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update "+payload.Resource.ResourceType, err)
			return
		}
		h.writeJSONResponse(w, http.StatusOK, obj)
	}

	switch payload.Resource.ResourceType {
	case "Deployment":
		var d appsv1.Deployment
		if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &d); err != nil {
			h.writeErrorResponse(w, http.StatusNotFound, "Deployment not found", err)
			return
		}
		updateWorkload(&d, d.Spec.Template.Spec.Containers, d.Spec.Replicas)

	case "ReplicaSet":
		var d appsv1.Deployment
		if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &d); err != nil {
			h.writeErrorResponse(w, http.StatusNotFound, "Deployment not found for ReplicaSet", err)
			return
		}
		updateWorkload(&d, d.Spec.Template.Spec.Containers, d.Spec.Replicas)

	case "StatefulSet":
		var s appsv1.StatefulSet
		if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &s); err != nil {
			h.writeErrorResponse(w, http.StatusNotFound, "StatefulSet not found", err)
			return
		}
		updateWorkload(&s, s.Spec.Template.Spec.Containers, s.Spec.Replicas)

	case "DaemonSet":
		var d appsv1.DaemonSet
		if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &d); err != nil {
			h.writeErrorResponse(w, http.StatusNotFound, "DaemonSet not found", err)
			return
		}
		updateWorkload(&d, d.Spec.Template.Spec.Containers, nil)

	default:
		h.writeErrorResponse(w, http.StatusBadRequest, "Unsupported resourceType", fmt.Errorf("got: %s", payload.Resource.ResourceType))
	}
}

// Напоминание: структура APIHandlers, NewAPIHandlers, writeJSONResponse, writeErrorResponse
// и webServerLog предполагаются определенными в другом файле этого пакета (например, handlers.go).

// === Pods ===
// Напоминание RBAC для kubedeck_controller.go:
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
func (h *KubedeckReconciler) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if data != nil {
		if err := json.NewEncoder(w).Encode(data); err != nil {
			webServerLog.Error(err, "Failed to write JSON response")
			// Заголовки уже отправлены, так что просто логируем
		}
	}
}

// Helper для записи ошибки
func (h *KubedeckReconciler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string, err error) {
	webServerLog.Error(err, message)
	response := metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status:  metav1.StatusFailure,
		Message: fmt.Sprintf("%s: %s", message, err.Error()),
		Reason:  metav1.StatusReason(http.StatusText(statusCode)), // Можно уточнить
		Code:    int32(statusCode),
	}
	h.writeJSONResponse(w, statusCode, response)
}

func (h *KubedeckReconciler) HandleCreatePod(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	if targetNamespace == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter for pod creation", fmt.Errorf("namespace is required"))
		return
	}

	var pod corev1.Pod
	if err := json.NewDecoder(req.Body).Decode(&pod); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for pod", err)
		return
	}
	defer req.Body.Close()

	pod.Namespace = targetNamespace
	pod.ResourceVersion = ""
	pod.UID = ""
	// pod.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod")) // Может быть полезно, если клиент не шлет GVK

	webServerLog.Info("Attempting to create pod", "namespace", pod.Namespace, "name", pod.GenerateName+pod.Name)
	if err := h.Client.Create(ctx, &pod); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create pod", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, pod)
}

func (h *KubedeckReconciler) HandleUpdatePod(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	podName := req.URL.Query().Get("name")

	if targetNamespace == "" || podName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name' query parameter for pod update", fmt.Errorf("namespace and name are required"))
		return
	}

	var updatedPod corev1.Pod
	if err := json.NewDecoder(req.Body).Decode(&updatedPod); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for pod update", err)
		return
	}
	defer req.Body.Close()

	if updatedPod.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'metadata.resourceVersion' in pod update request body", fmt.Errorf("resourceVersion is required for updates"))
		return
	}
	updatedPod.Namespace = targetNamespace
	updatedPod.Name = podName
	// updatedPod.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	webServerLog.Info("Attempting to update pod", "namespace", updatedPod.Namespace, "name", updatedPod.Name, "resourceVersion", updatedPod.ResourceVersion)
	if err := h.Client.Update(ctx, &updatedPod); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update pod", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, updatedPod)
}

func (h *KubedeckReconciler) HandleDeletePod(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	podName := req.URL.Query().Get("name")

	if targetNamespace == "" || podName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name' query parameter for pod deletion", fmt.Errorf("namespace and name are required"))
		return
	}

	podToDelete := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: targetNamespace,
		},
	}
	// podToDelete.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Pod"))

	webServerLog.Info("Attempting to delete pod", "namespace", targetNamespace, "name", podName)
	// Можно добавить &client.DeleteOptions{} для указания PropagationPolicy и др.
	if err := h.Client.Delete(ctx, podToDelete); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete pod", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "Pod", "name": podName, "namespace": targetNamespace})
}

// === ConfigMaps ===
// Напоминание RBAC для kubedeck_controller.go:
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

func (h *KubedeckReconciler) HandleCreateConfigMap(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	if targetNamespace == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter for ConfigMap creation", fmt.Errorf("namespace is required"))
		return
	}

	var cm corev1.ConfigMap
	if err := json.NewDecoder(req.Body).Decode(&cm); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for ConfigMap", err)
		return
	}
	defer req.Body.Close()
	cm.Namespace = targetNamespace
	cm.ResourceVersion = ""
	cm.UID = ""
	// cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	webServerLog.Info("Attempting to create ConfigMap", "namespace", cm.Namespace, "name", cm.Name)
	if err := h.Client.Create(ctx, &cm); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create ConfigMap", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, cm)
}

func (h *KubedeckReconciler) HandleUpdateConfigMap(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	cmName := req.URL.Query().Get("name")

	if targetNamespace == "" || cmName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name' query parameter for ConfigMap update", fmt.Errorf("namespace and name are required"))
		return
	}

	var updatedCm corev1.ConfigMap
	if err := json.NewDecoder(req.Body).Decode(&updatedCm); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for ConfigMap update", err)
		return
	}
	defer req.Body.Close()

	if updatedCm.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'metadata.resourceVersion' in ConfigMap update request", fmt.Errorf("resourceVersion is required"))
		return
	}
	updatedCm.Namespace = targetNamespace
	updatedCm.Name = cmName
	// updatedCm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	webServerLog.Info("Attempting to update ConfigMap", "namespace", updatedCm.Namespace, "name", updatedCm.Name)
	if err := h.Client.Update(ctx, &updatedCm); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update ConfigMap", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, updatedCm)
}

func (h *KubedeckReconciler) HandleDeleteConfigMap(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	cmName := req.URL.Query().Get("name")

	if targetNamespace == "" || cmName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name' query parameter for ConfigMap deletion", fmt.Errorf("namespace and name are required"))
		return
	}

	cmToDelete := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: targetNamespace},
	}
	// cmToDelete.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	webServerLog.Info("Attempting to delete ConfigMap", "namespace", targetNamespace, "name", cmName)
	if err := h.Client.Delete(ctx, cmToDelete); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete ConfigMap", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "ConfigMap", "name": cmName, "namespace": targetNamespace})
}

// === Deployments ===
// Напоминание RBAC для kubedeck_controller.go:
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (h *KubedeckReconciler) HandleCreateDeployment(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	if targetNamespace == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter for Deployment creation", fmt.Errorf("namespace is required"))
		return
	}

	var deploy appsv1.Deployment
	if err := json.NewDecoder(req.Body).Decode(&deploy); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for Deployment", err)
		return
	}
	defer req.Body.Close()
	deploy.Namespace = targetNamespace
	deploy.ResourceVersion = ""
	deploy.UID = ""
	// deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))

	webServerLog.Info("Attempting to create Deployment", "namespace", deploy.Namespace, "name", deploy.Name)
	if err := h.Client.Create(ctx, &deploy); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create Deployment", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, deploy)
}

func (h *KubedeckReconciler) HandleUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace string `json:"namespace"`
		Resource  struct {
			ResourceType  string `json:"resourceType"`
			Name          string `json:"name"`
			CPURequest    int    `json:"cpuRequest"`
			MemoryRequest int    `json:"memoryRequest"`
			CPULimit      int    `json:"cpuLimit"`
			MemoryLimit   int    `json:"memoryLimit"`
			Replicas      int32  `json:"replicas"`
		} `json:"resource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}
	defer r.Body.Close()

	// если resourceType == ReplicaSet, обрезаем name
	if payload.Resource.ResourceType == "ReplicaSet" {
		parts := strings.Split(payload.Resource.Name, "-")
		if len(parts) > 1 {
			payload.Resource.Name = strings.Join(parts[:len(parts)-1], "-")
		}
	}

	ctx := r.Context()
	var deployment appsv1.Deployment
	if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &deployment); err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "Deployment not found", err)
		return
	}
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		container := &deployment.Spec.Template.Spec.Containers[0]
		container.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPURequest)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryRequest)),
		}
		container.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPULimit)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryLimit)),
		}
	}
	deployment.Spec.Replicas = &payload.Resource.Replicas

	if err := h.Client.Update(ctx, &deployment); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update Deployment", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, deployment)
}

func (h *KubedeckReconciler) HandleDeleteDeployment(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	targetNamespace := req.URL.Query().Get("namespace")
	deployName := req.URL.Query().Get("name")

	if targetNamespace == "" || deployName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name' query parameter for Deployment deletion", fmt.Errorf("namespace and name are required"))
		return
	}

	deployToDelete := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: deployName, Namespace: targetNamespace},
	}
	// deployToDelete.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))

	webServerLog.Info("Attempting to delete Deployment", "namespace", targetNamespace, "name", deployName)
	if err := h.Client.Delete(ctx, deployToDelete); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete Deployment", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "Deployment", "name": deployName, "namespace": targetNamespace})
}

// === PersistentVolume, PV ===
func (h *KubedeckReconciler) HandleCreatePersistentVolume(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj corev1.PersistentVolume
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for PersistentVolume", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create PersistentVolume", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdatePersistentVolume(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	var obj corev1.PersistentVolume
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for PersistentVolume update", err)
		return
	}
	defer req.Body.Close()

	if obj.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing resourceVersion", fmt.Errorf("resourceVersion required"))
		return
	}

	obj.Namespace = ns
	obj.Name = name

	if err := h.Client.Update(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update PersistentVolume", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, obj)
}

func (h *KubedeckReconciler) HandleDeletePersistentVolume(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete PersistentVolume", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "PersistentVolume", "name": name, "namespace": ns})
}

// === Namespaces (Пример для кластерного ресурса) ===
// Напоминание RBAC для kubedeck_controller.go:
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
func (h *KubedeckReconciler) HandleCreateNamespace(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	var ns corev1.Namespace
	if err := json.NewDecoder(req.Body).Decode(&ns); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for Namespace", err)
		return
	}
	defer req.Body.Close()
	ns.ResourceVersion = ""
	ns.UID = ""
	// ns.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Namespace"))

	webServerLog.Info("Attempting to create Namespace", "name", ns.Name)
	if err := h.Client.Create(ctx, &ns); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create Namespace", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, ns)
}

func (h *KubedeckReconciler) HandleUpdateNamespace(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	nsName := req.URL.Query().Get("name") // Имя берем из query параметра
	if nsName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'name' query parameter for Namespace update", fmt.Errorf("name is required"))
		return
	}

	var updatedNs corev1.Namespace
	if err := json.NewDecoder(req.Body).Decode(&updatedNs); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON body for Namespace update", err)
		return
	}
	defer req.Body.Close()

	if updatedNs.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'metadata.resourceVersion' in Namespace update request", fmt.Errorf("resourceVersion is required"))
		return
	}
	updatedNs.Name = nsName // Убедимся, что имя из URL
	// updatedNs.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Namespace"))

	webServerLog.Info("Attempting to update Namespace", "name", updatedNs.Name)
	if err := h.Client.Update(ctx, &updatedNs); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update Namespace", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, updatedNs)
}

func (h *KubedeckReconciler) HandleDeleteNamespace(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	nsName := req.URL.Query().Get("name")
	if nsName == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'name' query parameter for Namespace deletion", fmt.Errorf("name is required"))
		return
	}

	nsToDelete := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: nsName},
	}
	// nsToDelete.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Namespace"))

	webServerLog.Info("Attempting to delete Namespace", "name", nsName)
	if err := h.Client.Delete(ctx, nsToDelete); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete Namespace", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "Namespace", "name": nsName})
}

// === DaemonSet ===
func (h *KubedeckReconciler) HandleCreateDaemonSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj appsv1.DaemonSet
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for DaemonSet", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create DaemonSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdateDaemonSet(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace string `json:"namespace"`
		Resource  struct {
			Name          string `json:"name"`
			CPURequest    int    `json:"cpuRequest"`
			MemoryRequest int    `json:"memoryRequest"`
			CPULimit      int    `json:"cpuLimit"`
			MemoryLimit   int    `json:"memoryLimit"`
		} `json:"resource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}
	defer r.Body.Close()

	ctx := r.Context()
	var ds appsv1.DaemonSet
	if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &ds); err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "DaemonSet not found", err)
		return
	}

	if len(ds.Spec.Template.Spec.Containers) > 0 {
		container := &ds.Spec.Template.Spec.Containers[0]
		container.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPURequest)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryRequest)),
		}
		container.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPULimit)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryLimit)),
		}
	}

	if err := h.Client.Update(ctx, &ds); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update DaemonSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, ds)
}

func (h *KubedeckReconciler) HandleDeleteDaemonSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete DaemonSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "DaemonSet", "name": name, "namespace": ns})
}

// === StatefulSet ===
func (h *KubedeckReconciler) HandleCreateStatefulSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj appsv1.StatefulSet
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for StatefulSet", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create StatefulSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdateStatefulSet(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace string `json:"namespace"`
		Resource  struct {
			Name          string `json:"name"`
			CPURequest    int    `json:"cpuRequest"`
			MemoryRequest int    `json:"memoryRequest"`
			CPULimit      int    `json:"cpuLimit"`
			MemoryLimit   int    `json:"memoryLimit"`
			Replicas      int32  `json:"replicas"`
		} `json:"resource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}
	defer r.Body.Close()

	ctx := r.Context()
	var sts appsv1.StatefulSet
	if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &sts); err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "StatefulSet not found", err)
		return
	}

	if len(sts.Spec.Template.Spec.Containers) > 0 {
		container := &sts.Spec.Template.Spec.Containers[0]
		container.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPURequest)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryRequest)),
		}
		container.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPULimit)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryLimit)),
		}
	}
	sts.Spec.Replicas = &payload.Resource.Replicas

	if err := h.Client.Update(ctx, &sts); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update StatefulSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, sts)
}

func (h *KubedeckReconciler) HandleDeleteStatefulSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete StatefulSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "StatefulSet", "name": name, "namespace": ns})
}

// === PersistentVolumeClaim, PVC ===
func (h *KubedeckReconciler) HandleCreatePersistentVolumeClaim(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj corev1.PersistentVolumeClaim
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for PersistentVolumeClaim", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create PersistentVolumeClaim", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdatePersistentVolumeClaim(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	var obj corev1.PersistentVolumeClaim
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for PersistentVolumeClaim update", err)
		return
	}
	defer req.Body.Close()

	if obj.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing resourceVersion", fmt.Errorf("resourceVersion required"))
		return
	}

	obj.Namespace = ns
	obj.Name = name

	if err := h.Client.Update(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update PersistentVolumeClaim", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, obj)
}

func (h *KubedeckReconciler) HandleDeletePersistentVolumeClaim(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete PersistentVolumeClaim", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "PersistentVolumeClaim", "name": name, "namespace": ns})
}

// === Service ===
func (h *KubedeckReconciler) HandleCreateService(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj corev1.Service
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for Service", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create Service", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdateService(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	var obj corev1.Service
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for Service update", err)
		return
	}
	defer req.Body.Close()

	if obj.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing resourceVersion", fmt.Errorf("resourceVersion required"))
		return
	}

	obj.Namespace = ns
	obj.Name = name

	if err := h.Client.Update(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update Service", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, obj)
}

func (h *KubedeckReconciler) HandleDeleteService(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete Service", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "Service", "name": name, "namespace": ns})
}

// === Ingress ===
func (h *KubedeckReconciler) HandleCreateIngress(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj networkingv1.Ingress
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for Ingress", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create Ingress", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdateIngress(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	var obj networkingv1.Ingress
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for Ingress update", err)
		return
	}
	defer req.Body.Close()

	if obj.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing resourceVersion", fmt.Errorf("resourceVersion required"))
		return
	}

	obj.Namespace = ns
	obj.Name = name

	if err := h.Client.Update(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update Ingress", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, obj)
}

func (h *KubedeckReconciler) HandleDeleteIngress(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete Ingress", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "Ingress", "name": name, "namespace": ns})
}

// === StorageClass ===
func (h *KubedeckReconciler) HandleCreateStorageClass(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var obj storagev1.StorageClass
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for StorageClass", err)
		return
	}
	defer req.Body.Close()

	obj.Namespace = ns
	obj.ResourceVersion = ""
	obj.UID = ""

	if err := h.Client.Create(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create StorageClass", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, obj)
}

func (h *KubedeckReconciler) HandleUpdateStorageClass(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	var obj storagev1.StorageClass
	if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for StorageClass update", err)
		return
	}
	defer req.Body.Close()

	if obj.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing resourceVersion", fmt.Errorf("resourceVersion required"))
		return
	}

	obj.Namespace = ns
	obj.Name = name

	if err := h.Client.Update(ctx, &obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update StorageClass", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, obj)
}

func (h *KubedeckReconciler) HandleDeleteStorageClass(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	obj := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, obj); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete StorageClass", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "StorageClass", "name": name, "namespace": ns})
}

// === Node ===
func (h *KubedeckReconciler) HandleUpdateNode(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	name := req.URL.Query().Get("name")
	if name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'name'", fmt.Errorf("name required"))
		return
	}

	var node corev1.Node
	if err := json.NewDecoder(req.Body).Decode(&node); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for Node update", err)
		return
	}
	defer req.Body.Close()

	if node.ResourceVersion == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing resourceVersion", fmt.Errorf("resourceVersion required"))
		return
	}

	node.Name = name

	if err := h.Client.Update(ctx, &node); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update Node", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, node)
}
func (h *KubedeckReconciler) HandleDeleteNode(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	name := req.URL.Query().Get("name")
	if name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'name'", fmt.Errorf("name required"))
		return
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	if err := h.Client.Delete(ctx, node); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete Node", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{"status": "deleted", "kind": "Node", "name": name})
}

// === ReplicaSet ===
func (h *KubedeckReconciler) HandleCreateReplicaSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	if ns == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' query parameter", fmt.Errorf("namespace required"))
		return
	}

	var rs appsv1.ReplicaSet
	if err := json.NewDecoder(req.Body).Decode(&rs); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON for ReplicaSet", err)
		return
	}
	defer req.Body.Close()

	rs.Namespace = ns
	rs.ResourceVersion = ""
	rs.UID = ""

	if err := h.Client.Create(ctx, &rs); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to create ReplicaSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusCreated, rs)
}

func (h *KubedeckReconciler) HandleUpdateReplicaSet(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Namespace string `json:"namespace"`
		Resource  struct {
			Name          string `json:"name"`
			CPURequest    int    `json:"cpuRequest"`
			MemoryRequest int    `json:"memoryRequest"`
			CPULimit      int    `json:"cpuLimit"`
			MemoryLimit   int    `json:"memoryLimit"`
			Replicas      int32  `json:"replicas"`
		} `json:"resource"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}
	defer r.Body.Close()

	ctx := r.Context()
	var rs appsv1.ReplicaSet
	if err := h.Client.Get(ctx, client.ObjectKey{Namespace: payload.Namespace, Name: payload.Resource.Name}, &rs); err != nil {
		h.writeErrorResponse(w, http.StatusNotFound, "ReplicaSet not found", err)
		return
	}
	if len(rs.Spec.Template.Spec.Containers) > 0 {
		container := &rs.Spec.Template.Spec.Containers[0]
		container.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPURequest)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryRequest)),
		}
		container.Resources.Limits = corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", payload.Resource.CPULimit)),
			corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", payload.Resource.MemoryLimit)),
		}
	}
	rs.Spec.Replicas = &payload.Resource.Replicas

	if err := h.Client.Update(ctx, &rs); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to update ReplicaSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, rs)
}

func (h *KubedeckReconciler) HandleDeleteReplicaSet(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	ns := req.URL.Query().Get("namespace")
	name := req.URL.Query().Get("name")
	if ns == "" || name == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing 'namespace' or 'name'", fmt.Errorf("namespace and name required"))
		return
	}

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
	}
	if err := h.Client.Delete(ctx, rs); err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete ReplicaSet", err)
		return
	}
	h.writeJSONResponse(w, http.StatusOK, map[string]string{
		"status":    "deleted",
		"kind":      "ReplicaSet",
		"name":      name,
		"namespace": ns,
	})
}
