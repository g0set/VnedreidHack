package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	LLMApiURL = "https://llm.glowbyteconsulting.com/api/chat/completions"
	LLMModel  = "anthropic.claude-sonnet-4-20250514"
)

// PodResourceInfo represents resource information for a pod
type PodResourceInfo struct {
	Name          string  `json:"name"`
	ResourceName  string  `json:"resourceName,omitempty"`
	ResourceType  string  `json:"resourceType,omitempty"`
	CPURequest    int     `json:"cpuRequest"`
	MemoryRequest int     `json:"memoryRequest"`
	CPULimit      int     `json:"cpuLimit"`
	MemoryLimit   int     `json:"memoryLimit"`
	CPUUsage      string  `json:"cpuUsage,omitempty"`
	MemoryUsage   string  `json:"memoryUsage,omitempty"`
	CPUPercentage float64 `json:"cpuPercentage,omitempty"`
	MemPercentage float64 `json:"memPercentage,omitempty"`
	Status        string  `json:"status,omitempty"`
	Replicas      int     `json:"replicas,omitempty"`
}

// LLMRequest represents the request structure for the LLM API
type LLMRequest struct {
	Model    string       `json:"model"`
	Messages []LLMMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

// LLMMessage represents a message in the LLM conversation
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents the response from the LLM API
type LLMResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Index        int    `json:"index"`
		Message      struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
}

// ResourceRecommendationResponse represents the response with resource recommendations
type ResourceRecommendationResponse struct {
	Message    string                       `json:"message"`
	Namespaces map[string][]PodResourceInfo `json:"namespaces"`
}

// handleResourceAnalysisRequest handles the request to analyze pod resource usage
func (r *KubedeckReconciler) handleResourceAnalysisRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	log := webServerLog.WithName("handleResourceAnalysisRequest")

	// Collect pod resource data
	podResourceData, err := r.collectPodResourceData(ctx)
	if err != nil {
		log.Error(err, "Failed to collect pod resource data")
		http.Error(w, "Failed to collect pod resource data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Send data to LLM for analysis
	recommendation, err := r.getLLMResourceRecommendations(ctx, podResourceData)
	if err != nil {
		log.Error(err, "Failed to get LLM recommendations")
		http.Error(w, "Failed to get LLM recommendations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the recommendation
	writeJsonResponse(w, recommendation, "resource recommendations", nil)
}

// collectPodResourceData collects resource data for all pods in the cluster
func (r *KubedeckReconciler) collectPodResourceData(ctx context.Context) (map[string][]PodResourceInfo, error) {
	// Get all pods
	var podList corev1.PodList
	if err := r.Client.List(ctx, &podList, &client.ListOptions{}); err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Get metrics client
	metricsClient, err := metricsv.NewForConfig(r.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics client: %w", err)
	}

	// Get pod metrics
	podMetricsList, err := metricsClient.MetricsV1beta1().PodMetricses(corev1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	// Create a map of pod metrics for quick lookup
	podMetricsMap := make(map[string]v1beta1.PodMetrics)
	for _, metric := range podMetricsList.Items {
		key := fmt.Sprintf("%s/%s", metric.Namespace, metric.Name)
		podMetricsMap[key] = metric
	}

	// Organize pod data by namespace
	result := make(map[string][]PodResourceInfo)

	for _, pod := range podList.Items {
		// Skip pods that are not Running
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}

		// Calculate total CPU and memory requests and limits
		var totalCPURequest, totalMemRequest, totalCPULimit, totalMemLimit resource.Quantity

		for _, container := range pod.Spec.Containers {
			if cpu, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
				totalCPURequest.Add(cpu)
			}
			if mem, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
				totalMemRequest.Add(mem)
			}
			if cpu, ok := container.Resources.Limits[corev1.ResourceCPU]; ok {
				totalCPULimit.Add(cpu)
			}
			if mem, ok := container.Resources.Limits[corev1.ResourceMemory]; ok {
				totalMemLimit.Add(mem)
			}
		}

		// Convert CPU millicores and memory to reasonable units
		cpuRequestMillicores := totalCPURequest.MilliValue()
		memoryRequestMi := totalMemRequest.Value() / (1024 * 1024) // Convert to Mi
		cpuLimitMillicores := totalCPULimit.MilliValue()
		memoryLimitMi := totalMemLimit.Value() / (1024 * 1024) // Convert to Mi

		// Get resource name and type (deployment, statefulset, etc)
		resourceName := ""
		resourceType := ""
		for _, ownerRef := range pod.OwnerReferences {
			resourceName = ownerRef.Name
			resourceType = ownerRef.Kind
			break
		}

		// Create pod resource info
		podInfo := PodResourceInfo{
			Name:          pod.Name,
			ResourceName:  resourceName,
			ResourceType:  resourceType,
			CPURequest:    int(cpuRequestMillicores),
			MemoryRequest: int(memoryRequestMi),
			CPULimit:      int(cpuLimitMillicores),
			MemoryLimit:   int(memoryLimitMi),
			Replicas:      -1, // Default value if not changed
		}

		// Add usage data if available
		metricKey := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)
		if metric, exists := podMetricsMap[metricKey]; exists {
			var totalCPUUsage, totalMemUsage resource.Quantity

			for _, container := range metric.Containers {
				totalCPUUsage.Add(container.Usage.Cpu().DeepCopy())
				totalMemUsage.Add(container.Usage.Memory().DeepCopy())
			}

			// Calculate usage percentages
			if !totalCPULimit.IsZero() {
				podInfo.CPUPercentage = (totalCPUUsage.AsApproximateFloat64() / totalCPULimit.AsApproximateFloat64()) * 100
				podInfo.CPUUsage = totalCPUUsage.String()
			}

			if !totalMemLimit.IsZero() {
				podInfo.MemPercentage = (totalMemUsage.AsApproximateFloat64() / totalMemLimit.AsApproximateFloat64()) * 100
				podInfo.MemoryUsage = totalMemUsage.String()
			}
		}

		// Add to result
		if _, exists := result[pod.Namespace]; !exists {
			result[pod.Namespace] = []PodResourceInfo{}
		}
		result[pod.Namespace] = append(result[pod.Namespace], podInfo)
	}

	return result, nil
}

// getLLMResourceRecommendations sends pod resource data to the LLM and gets recommendations
func (r *KubedeckReconciler) getLLMResourceRecommendations(ctx context.Context, podResourceData map[string][]PodResourceInfo) (*ResourceRecommendationResponse, error) {
	log := webServerLog.WithName("getLLMResourceRecommendations")

	// Create the prompt for the LLM
	prompt := r.createLLMPrompt(podResourceData)

	// Create the LLM request
	llmReq := LLMRequest{
		Model: LLMModel,
		Messages: []LLMMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	// Marshal the request
	reqBody, err := json.Marshal(llmReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal LLM request: %w", err)
	}

	// Create HTTP request with timeout
	httpCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", LLMApiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer sk-2511d8de29004787aff3b7f83bab43db")

	// Send the request
	log.Info("Sending request to LLM API", "url", LLMApiURL)
	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to LLM API: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		var errorResponse map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return nil, fmt.Errorf("LLM API returned status %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("LLM API returned status %d: %v", resp.StatusCode, errorResponse)
	}

	// Parse the response
	var llmResp LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return nil, fmt.Errorf("failed to decode LLM response: %w", err)
	}

	// Check if the response has at least one choice
	if len(llmResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM response has no choices")
	}

	// Extract the content
	content := llmResp.Choices[0].Message.Content
	log.Info("Received response from LLM", "content", content)

	// Parse the LLM response to get the recommendations
	var recommendation ResourceRecommendationResponse
	if err := json.Unmarshal([]byte(content), &recommendation); err != nil {
		// If we can't parse the JSON directly, try to extract it from the content
		jsonStart := strings.Index(content, "{")
		jsonEnd := strings.LastIndex(content, "}")
		if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
			jsonContent := content[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonContent), &recommendation); err != nil {
				return nil, fmt.Errorf("failed to extract and parse recommendation JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse recommendation: %w", err)
		}
	}

	return &recommendation, nil
}

// createLLMPrompt creates the prompt for the LLM based on pod resource data
func (r *KubedeckReconciler) createLLMPrompt(podResourceData map[string][]PodResourceInfo) string {
	sb := strings.Builder{}

	// Получаем текущий стиль ответов
	responseStyle := r.TelegramBotSettings.GetResponseStyle()

	// Если стиль не задан, используем стиль по умолчанию
	if responseStyle == "" {
		responseStyle = "Проанализируйте ресурсы подов Kubernetes и предоставьте технически обоснованные рекомендации."
	}

	sb.WriteString(responseStyle + " ")
	sb.WriteString("Укажите ТОЛЬКО те поды, которые имеют ЗНАЧИТЕЛЬНЫЕ отклонения в потреблении ресурсов. ")
	sb.WriteString("Ответ ТОЛЬКО в следующем JSON формате:\n\n")

	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"message\": \"Суть анализа\",\n")
	sb.WriteString("  \"namespaces\": {\n")
	sb.WriteString("    \"имя-пространства\": [\n")
	sb.WriteString("      {\n")
	sb.WriteString("        \"name\": \"имя-пода\",\n")
	sb.WriteString("        \"resourceName\": \"имя-ресурса\",\n")
	sb.WriteString("        \"resourceType\": \"тип-ресурса\",\n")
	sb.WriteString("        \"cpuRequest\": 300,  // CPU request в милликорах или -1 если не нужно менять\n")
	sb.WriteString("        \"memoryRequest\": 64,  // Memory request в Mi или -1 если не нужно менять\n")
	sb.WriteString("        \"cpuLimit\": 500,  // CPU limit в милликорах или -1 если не нужно менять\n")
	sb.WriteString("        \"memoryLimit\": 128,  // Memory limit в Mi или -1 если не нужно менять\n")
	sb.WriteString("        \"replicas\": 3,  // Количество реплик или -1 если не нужно менять\n")
	sb.WriteString("        \"status\": \"warning\" или \"critical\"\n")
	sb.WriteString("      }\n")
	sb.WriteString("    ]\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("Критерии для анализа:\n")
	sb.WriteString("1. CPU <25% - рекомендуется снизить (warning), только при значительной разнице\n")
	sb.WriteString("2. CPU >85% - необходимо повысить (critical)\n")
	sb.WriteString("3. Память <30% - рекомендуется снизить (warning), только при значительной разнице\n")
	sb.WriteString("4. Память >80% - необходимо повысить (critical)\n")
	sb.WriteString("5. Если не требуется изменений - установите значение -1\n")
	sb.WriteString("6. Указывайте только поды с СУЩЕСТВЕННЫМИ проблемами, игнорируйте незначительные отклонения\n")
	sb.WriteString("7. Для каждого пода указывайте как requests, так и limits\n")
	sb.WriteString("8. В поле resourceType указывайте тип ресурса (Deployment, StatefulSet, ReplicaSet, ...)\n")
	sb.WriteString("9. В поле replicas указывайте рекомендуемое количество реплик или -1, если изменение не требуется\n\n")

	sb.WriteString("Данные о подах:\n\n")

	// Add pod resource data to the prompt
	for namespace, pods := range podResourceData {
		sb.WriteString(fmt.Sprintf("Пространство имен: %s\n", namespace))
		for _, pod := range pods {
			sb.WriteString(fmt.Sprintf("- Под: %s\n", pod.Name))
			if pod.ResourceName != "" {
				sb.WriteString(fmt.Sprintf("  Ресурс: %s\n", pod.ResourceName))
			}
			if pod.ResourceType != "" {
				sb.WriteString(fmt.Sprintf("  Тип ресурса: %s\n", pod.ResourceType))
			}
			sb.WriteString(fmt.Sprintf("  CPU Request: %dm, CPU Limit: %dm, ", pod.CPURequest, pod.CPULimit))
			if pod.CPUUsage != "" {
				sb.WriteString(fmt.Sprintf("Потребление: %s (%.1f%%)\n", pod.CPUUsage, pod.CPUPercentage))
			} else {
				sb.WriteString("Потребление: неизвестно\n")
			}

			sb.WriteString(fmt.Sprintf("  Memory Request: %dMi, Memory Limit: %dMi, ", pod.MemoryRequest, pod.MemoryLimit))
			if pod.MemoryUsage != "" {
				sb.WriteString(fmt.Sprintf("Потребление: %s (%.1f%%)\n", pod.MemoryUsage, pod.MemPercentage))
			} else {
				sb.WriteString("Потребление: неизвестно\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Для корректного ресайзинга ресурсов обязательно указывайте resourceName и resourceType для каждого пода.\n\n")
	sb.WriteString("Для каждого пода необходимо возвращать как requests, так и limits, то есть как минимальные необходимые ресурсы, так и максимальные.\n\n")
	sb.WriteString("Рекомендуемое количество реплик возвращайте только при наличии веских причин для изменения.\n\n")
	sb.WriteString("В поле \"message\" укажите для каждого пода точные процентные значения текущего использования ресурсов (CPU и Memory) и конкретные рекомендуемые значения ресурсов в числовом виде. Например: \"Под nginx-123 использует 95% CPU (950m из 1000m), рекомендуется увеличить до 1500m\".\n\n")
	sb.WriteString("Верните только JSON без дополнительных пояснений. Ответ должен быть на русском языке в профессиональном техническом стиле.")

	return sb.String()
}
