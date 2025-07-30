package controller

import (
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeMetrics представляет метрики узла
type NodeMetrics struct {
	Name          string  `json:"name"`
	CPU           string  `json:"cpu"`
	CPUPercentage float64 `json:"cpuPercentage"`
	Memory        string  `json:"memory"`
	MemPercentage float64 `json:"memPercentage"`
	CPUCapacity   string  `json:"cpuCapacity"`
	MemCapacity   string  `json:"memCapacity"`
}

// PodMetrics представляет метрики пода
type PodMetrics struct {
	Name          string  `json:"name"`
	Namespace     string  `json:"namespace"`
	CPU           string  `json:"cpu"`
	Memory        string  `json:"memory"`
	CPUPercentage float64 `json:"cpuPercentage"`
	MemPercentage float64 `json:"memPercentage"`
}

// ClusterMetrics представляет общие метрики кластера
type ClusterMetrics struct {
	TotalCPU          string        `json:"totalCPU"`
	TotalMemory       string        `json:"totalMemory"`
	UsedCPU           string        `json:"usedCPU"`
	UsedMemory        string        `json:"usedMemory"`
	CPUUtilization    float64       `json:"cpuUtilization"`
	MemoryUtilization float64       `json:"memoryUtilization"`
	Nodes             []NodeMetrics `json:"nodes,omitempty"`
	NodeCount         int           `json:"nodeCount"`
	ReadyNodeCount    int           `json:"readyNodeCount"`
}

// getMetricsClient создает клиент для доступа к Metrics API
func (r *KubedeckReconciler) getMetricsClient() (*metricsv.Clientset, error) {
	return metricsv.NewForConfig(r.Config)
}

// handleClusterMetricsRequest обрабатывает запрос на получение метрик кластера
func (r *KubedeckReconciler) handleClusterMetricsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := webServerLog.WithName("handleClusterMetricsRequest")

	// Получаем параметры запроса
	includeNodesStr := req.URL.Query().Get("includeNodes")
	includeNodes := includeNodesStr == "true"

	// Получаем метрики узлов
	metricsClient, err := r.getMetricsClient()
	if err != nil {
		log.Error(err, "Failed to create metrics client")
		http.Error(w, "Failed to create metrics client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем список узлов
	nodeList := &corev1.NodeList{}
	err = r.Client.List(ctx, nodeList, &client.ListOptions{})
	if err != nil {
		log.Error(err, "Failed to list nodes")
		http.Error(w, "Failed to list nodes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем метрики узлов
	nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Error(err, "Failed to list node metrics")
		http.Error(w, "Failed to list node metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаем карту метрик узлов для быстрого доступа
	nodeMetricsMap := make(map[string]v1beta1.NodeMetrics, len(nodeMetricsList.Items))
	for _, metrics := range nodeMetricsList.Items {
		nodeMetricsMap[metrics.Name] = metrics
	}

	var totalCPUCapacity, totalMemoryCapacity resource.Quantity
	var totalCPUUsed, totalMemoryUsed resource.Quantity

	nodeMetrics := make([]NodeMetrics, 0, len(nodeList.Items))
	readyNodeCount := 0

	for _, node := range nodeList.Items {
		// Проверяем готовность узла
		if isNodeReady(node) {
			readyNodeCount++
		}

		// Пропускаем узлы с NoSchedule
		if hasNoScheduleTaint(node) {
			continue
		}

		// Получаем емкость узла
		cpuCapacity := node.Status.Capacity.Cpu().DeepCopy()
		memoryCapacity := node.Status.Capacity.Memory().DeepCopy()

		totalCPUCapacity.Add(cpuCapacity)
		totalMemoryCapacity.Add(memoryCapacity)

		// Проверяем наличие метрик
		metrics, exists := nodeMetricsMap[node.Name]
		if !exists {
			nodeMetrics = append(nodeMetrics, NodeMetrics{
				Name:        node.Name,
				CPU:         "0",
				Memory:      "0",
				CPUCapacity: cpuCapacity.String(),
				MemCapacity: memoryCapacity.String(),
			})
			continue
		}

		// Получаем использование
		cpuUsed := metrics.Usage.Cpu().DeepCopy()
		memoryUsed := metrics.Usage.Memory().DeepCopy()

		totalCPUUsed.Add(cpuUsed)
		totalMemoryUsed.Add(memoryUsed)

		// Вычисляем процент использования
		cpuPercentage := calculateUtilization(cpuUsed, cpuCapacity)
		memPercentage := calculateUtilization(memoryUsed, memoryCapacity)

		nodeMetrics = append(nodeMetrics, NodeMetrics{
			Name:          node.Name,
			CPU:           cpuUsed.String(),
			Memory:        memoryUsed.String(),
			CPUPercentage: cpuPercentage,
			MemPercentage: memPercentage,
			CPUCapacity:   cpuCapacity.String(),
			MemCapacity:   memoryCapacity.String(),
		})
	}

	// Вычисляем процент использования для кластера
	cpuUtilization := calculateUtilization(totalCPUUsed, totalCPUCapacity)
	memoryUtilization := calculateUtilization(totalMemoryUsed, totalMemoryCapacity)

	// Создаем объект метрик кластера
	result := ClusterMetrics{
		TotalCPU:          totalCPUCapacity.String(),
		TotalMemory:       totalMemoryCapacity.String(),
		UsedCPU:           totalCPUUsed.String(),
		UsedMemory:        totalMemoryUsed.String(),
		CPUUtilization:    cpuUtilization,
		MemoryUtilization: memoryUtilization,
		NodeCount:         len(nodeList.Items),
		ReadyNodeCount:    readyNodeCount,
	}

	// Включаем информацию о узлах только если запрошено
	if includeNodes {
		result.Nodes = nodeMetrics
	}

	writeJsonResponse(w, result, "cluster metrics", nil)
}

// handlePodMetricsRequest обрабатывает запрос на получение метрик подов
func (r *KubedeckReconciler) handlePodMetricsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := webServerLog.WithName("handlePodMetricsRequest")

	// Получаем параметры запроса
	namespace := req.URL.Query().Get("namespace")
	podName := req.URL.Query().Get("name")

	// Получаем клиент для метрик
	metricsClient, err := r.getMetricsClient()
	if err != nil {
		log.Error(err, "Failed to create metrics client")
		http.Error(w, "Failed to create metrics client: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем список подов для определения лимитов
	var podList corev1.PodList
	if namespace != "" {
		err = r.Client.List(ctx, &podList, &client.ListOptions{Namespace: namespace})
	} else {
		err = r.Client.List(ctx, &podList, &client.ListOptions{})
	}
	if err != nil {
		log.Error(err, "Failed to list pods")
		http.Error(w, "Failed to list pods: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Создаем карту подов для быстрого доступа к лимитам
	podLimitsMap := make(map[string]map[string]corev1.ResourceList)
	for _, pod := range podList.Items {
		key := pod.Namespace + "/" + pod.Name
		podLimitsMap[key] = make(map[string]corev1.ResourceList)

		for _, container := range pod.Spec.Containers {
			podLimitsMap[key][container.Name] = container.Resources.Limits
		}
	}

	// Если запрошен конкретный под
	if podName != "" && namespace != "" {
		podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			log.Error(err, "Failed to get pod metrics", "pod", podName, "namespace", namespace)
			http.Error(w, "Failed to get pod metrics: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Суммируем использование CPU и памяти по всем контейнерам
		var totalCPU, totalMemory resource.Quantity
		var totalCPULimit, totalMemoryLimit resource.Quantity
		key := namespace + "/" + podName

		for _, container := range podMetrics.Containers {
			totalCPU.Add(container.Usage.Cpu().DeepCopy())
			totalMemory.Add(container.Usage.Memory().DeepCopy())

			// Добавляем лимиты, если они определены
			if limits, exists := podLimitsMap[key][container.Name]; exists {
				if cpuLimit, ok := limits[corev1.ResourceCPU]; ok {
					totalCPULimit.Add(cpuLimit)
				}
				if memLimit, ok := limits[corev1.ResourceMemory]; ok {
					totalMemoryLimit.Add(memLimit)
				}
			}
		}

		// Вычисляем проценты использования
		cpuPercentage := calculateUtilization(totalCPU, totalCPULimit)
		memPercentage := calculateUtilization(totalMemory, totalMemoryLimit)

		result := PodMetrics{
			Name:          podMetrics.Name,
			Namespace:     podMetrics.Namespace,
			CPU:           totalCPU.String(),
			Memory:        totalMemory.String(),
			CPUPercentage: cpuPercentage,
			MemPercentage: memPercentage,
		}

		writeJsonResponse(w, result, "pod metrics", nil)
		return
	}

	// Если запрошены все поды в определенном namespace или во всех namespace
	var podMetricsList *v1beta1.PodMetricsList
	if namespace != "" {
		podMetricsList, err = metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{})
	} else {
		podMetricsList, err = metricsClient.MetricsV1beta1().PodMetricses(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	}

	if err != nil {
		log.Error(err, "Failed to list pod metrics", "namespace", namespace)
		http.Error(w, "Failed to list pod metrics: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Преобразуем метрики в удобный формат
	results := make([]PodMetrics, 0, len(podMetricsList.Items))
	for _, podMetrics := range podMetricsList.Items {
		var totalCPU, totalMemory resource.Quantity
		var totalCPULimit, totalMemoryLimit resource.Quantity
		key := podMetrics.Namespace + "/" + podMetrics.Name

		for _, container := range podMetrics.Containers {
			totalCPU.Add(container.Usage.Cpu().DeepCopy())
			totalMemory.Add(container.Usage.Memory().DeepCopy())

			// Добавляем лимиты, если они определены
			if limits, exists := podLimitsMap[key][container.Name]; exists {
				if cpuLimit, ok := limits[corev1.ResourceCPU]; ok {
					totalCPULimit.Add(cpuLimit)
				}
				if memLimit, ok := limits[corev1.ResourceMemory]; ok {
					totalMemoryLimit.Add(memLimit)
				}
			}
		}

		// Вычисляем проценты использования
		cpuPercentage := calculateUtilization(totalCPU, totalCPULimit)
		memPercentage := calculateUtilization(totalMemory, totalMemoryLimit)

		results = append(results, PodMetrics{
			Name:          podMetrics.Name,
			Namespace:     podMetrics.Namespace,
			CPU:           totalCPU.String(),
			Memory:        totalMemory.String(),
			CPUPercentage: cpuPercentage,
			MemPercentage: memPercentage,
		})
	}

	writeJsonResponse(w, results, "pod metrics", nil)
}

// isNodeReady проверяет, готов ли узел
func isNodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

// hasNoScheduleTaint проверяет, имеет ли узел taint NoSchedule
func hasNoScheduleTaint(node corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectNoSchedule {
			return true
		}
	}
	return false
}

// calculateUtilization вычисляет процент использования ресурса
func calculateUtilization(used, capacity resource.Quantity) float64 {
	if capacity.IsZero() {
		return 0
	}

	usedValue := used.AsApproximateFloat64()
	capacityValue := capacity.AsApproximateFloat64()

	return (usedValue / capacityValue) * 100
}
