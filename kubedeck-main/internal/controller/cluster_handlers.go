package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

type ClusterProvider interface {
	ListClusters(ctx context.Context) ([]Cluster, error)
	GetNodeGroups(ctx context.Context, clusterID string) ([]NodeGroup, error)
	ScaleNodeGroup(ctx context.Context, clusterID, groupID string, nodeCount int) error
}

type Cluster struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Status   string `json:"status"`
	Provider string `json:"provider"`
}

type NodeGroup struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NodeCount int    `json:"node_count"`
}

// TimeWeb provider
type TimeWebProvider struct {
	token   string
	baseURL string
	client  *http.Client
}

// TimeWeb API response structures
type TimeWebClusterResponse struct {
	ResponseID string `json:"response_id"`
	Meta       struct {
		Total int `json:"total"`
	} `json:"meta"`
	Clusters []struct {
		ID            int    `json:"id"`
		Name          string `json:"name"`
		CreatedAt     string `json:"created_at"`
		Status        string `json:"status"`
		Description   string `json:"description"`
		HA            bool   `json:"ha"`
		K8sVersion    string `json:"k8s_version"`
		AvatarLink    string `json:"avatar_link"`
		NetworkDriver string `json:"network_driver"`
		Ingress       bool   `json:"ingress"`
		PresetID      int    `json:"preset_id"`
		CPU           int    `json:"cpu"`
		RAM           int    `json:"ram"`
		Disk          int    `json:"disk"`
	} `json:"clusters"`
}

// TimeWeb API response structure для групп нод
type TimeWebNodeGroupResponse struct {
	ResponseID string `json:"response_id"`
	Meta       struct {
		Total int `json:"total"`
	} `json:"meta"`
	NodeGroups []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		CreatedAt string `json:"created_at"`
		PresetID  int    `json:"preset_id"`
		NodeCount int    `json:"node_count"`
	} `json:"node_groups"`
}

func NewTimeWebProvider() *TimeWebProvider {
	return &TimeWebProvider{
		token:   "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCIsImtpZCI6IjFrYnhacFJNQGJSI0tSbE1xS1lqIn0.eyJ1c2VyIjoiY2E0ODMzNiIsInR5cGUiOiJhcGlfa2V5IiwiYXBpX2tleV9pZCI6Ijk2NDU3YmM5LTU5MTEtNGI4OC04YWM0LTEwYWYwNTJkYjllYiIsImlhdCI6MTc0OTM3NjQ4NSwiZXhwIjoxNzUxOTY4NDg0fQ.A8e6SZGnOzi9SnNogZ8TWAClZ5kDMjcIrPBmhzBBV8kpThUmPqD0XGYwYqPk1xQMfQ6BJ0Z0rBm52woGzQvrul09RaNuGe6LX1IuPy7F6H4b7h0Nd-91m-nf2vHldmIeNeOGv41iQILItnQxNJlbrAjrmIMMVV7C8rP4I5j57GblpZ98yU6Ak2H3e-zL3Aor2ZB_ftdCYRC4T4R9RouCDuUXHNW-OV_fq_Y8NtalPp8drXhuFZb5Q4WombOFyytG5qLP_nIlAAdQuXIjeFHeMbjf8fg7Ra13yt5hhqBrmZGj27U9UvMnZILpFrcgug_cJJxuSq-WRIMwtXMg2_XiX0PG2LOA3khbpcMvdGMilGSL09KEDQDwzpTZmIwzqRGMGn32ugWDqhhVyxcUpaUuO7MoBCfjG9njaCF72lhglVuIIpgJpywU83S65QkTVJUPQ-zF6UzFlKR5VTDg0DSThl9LP9BTjNj6_O0l4rGI3epjbGg6fD9STeQQihmX8xFb",
		baseURL: "https://api.timeweb.cloud/api/v1",
		client:  &http.Client{},
	}
}

func (t *TimeWebProvider) ListClusters(ctx context.Context) ([]Cluster, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", t.baseURL+"/k8s/clusters", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result TimeWebClusterResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	clusters := make([]Cluster, len(result.Clusters))
	for i, c := range result.Clusters {
		clusters[i] = Cluster{
			ID:       strconv.Itoa(c.ID),
			Name:     c.Name,
			Status:   c.Status,
			Provider: "timeweb",
		}
	}

	return clusters, nil
}

func (t *TimeWebProvider) GetNodeGroups(ctx context.Context, clusterID string) ([]NodeGroup, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/k8s/clusters/%s/groups", t.baseURL, clusterID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result TimeWebNodeGroupResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	groups := make([]NodeGroup, len(result.NodeGroups))
	for i, g := range result.NodeGroups {
		groups[i] = NodeGroup{
			ID:        strconv.Itoa(g.ID),
			Name:      g.Name,
			NodeCount: g.NodeCount,
		}
	}

	return groups, nil
}

func (t *TimeWebProvider) ScaleNodeGroup(ctx context.Context, clusterID, groupID string, nodeCount int) error {
	data, _ := json.Marshal(map[string]int{"count": nodeCount})

	method := "POST"
	if nodeCount < 0 {
		method = "DELETE"
		nodeCount = -nodeCount
		data, _ = json.Marshal(map[string]int{"count": nodeCount})
	}

	req, err := http.NewRequestWithContext(ctx, method,
		fmt.Sprintf("%s/k8s/clusters/%s/groups/%s/nodes", t.baseURL, clusterID, groupID),
		bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.token)

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scale request failed: %s", string(body))
	}

	return nil
}

// Yandex Cloud provider остается без изменений
type YandexCloudProvider struct {
	token    string
	folderID string
	baseURL  string
	client   *http.Client
}

func NewYandexCloudProvider() *YandexCloudProvider {
	return &YandexCloudProvider{
		token:    os.Getenv("YANDEX_CLOUD_TOKEN"),
		folderID: os.Getenv("YANDEX_CLOUD_FOLDER_ID"),
		baseURL:  "https://mks.api.cloud.yandex.net/managed-kubernetes/v1",
		client:   &http.Client{},
	}
}

func (y *YandexCloudProvider) ListClusters(ctx context.Context) ([]Cluster, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/clusters?folderId=%s", y.baseURL, y.folderID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+y.token)

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Clusters []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"clusters"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	clusters := make([]Cluster, len(result.Clusters))
	for i, c := range result.Clusters {
		clusters[i] = Cluster{
			ID:       c.ID,
			Name:     c.Name,
			Status:   c.Status,
			Provider: "yandex",
		}
	}

	return clusters, nil
}

func (y *YandexCloudProvider) GetNodeGroups(ctx context.Context, clusterID string) ([]NodeGroup, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s/nodeGroups?clusterId=%s", y.baseURL, clusterID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+y.token)

	resp, err := y.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		NodeGroups []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			ScalePolicy struct {
				FixedScale struct {
					Size int `json:"size"`
				} `json:"fixedScale"`
			} `json:"scalePolicy"`
		} `json:"nodeGroups"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	groups := make([]NodeGroup, len(result.NodeGroups))
	for i, g := range result.NodeGroups {
		groups[i] = NodeGroup{
			ID:        g.ID,
			Name:      g.Name,
			NodeCount: g.ScalePolicy.FixedScale.Size,
		}
	}

	return groups, nil
}

func (y *YandexCloudProvider) ScaleNodeGroup(ctx context.Context, clusterID, groupID string, nodeCount int) error {
	data, _ := json.Marshal(map[string]interface{}{
		"updateMask": "scalePolicy.fixedScale.size",
		"scalePolicy": map[string]interface{}{
			"fixedScale": map[string]int{
				"size": nodeCount,
			},
		},
	})

	req, err := http.NewRequestWithContext(ctx, "PATCH",
		fmt.Sprintf("%s/nodeGroups/%s", y.baseURL, groupID),
		bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+y.token)

	resp, err := y.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scale request failed: %s", string(body))
	}

	return nil
}

// Factory
func NewClusterProvider(providerType string) (ClusterProvider, error) {
	switch providerType {
	case "timeweb":
		return NewTimeWebProvider(), nil
	case "yandex":
		return NewYandexCloudProvider(), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerType)
	}
}
