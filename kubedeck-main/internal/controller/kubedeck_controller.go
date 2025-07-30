package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrlv1 "nikcorp.ru/kubedeck/api/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// KubedeckReconciler reconciles a Kubedeck object
type KubedeckReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config *rest.Config
	// Cloud providers
	timewebProvider     ClusterProvider
	yandexCloudProvider ClusterProvider
	TelegramBotSettings *TelegramBotSettings
}

// Separate logger for the web server
var webServerLog = logf.Log.WithName("kubedeck-webserver")

// +kubebuilder:rbac:groups=*,resources=*,verbs=*
// +kubebuilder:rbac:groups=ctrl.nikcorp.ru,resources=kubedecks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ctrl.nikcorp.ru,resources=kubedecks/finalizers,verbs=update
func (r *KubedeckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)
	// TODO(user): your logic here
	return ctrl.Result{}, nil
}

// --- Handler Helper ---
func writeJsonResponse(w http.ResponseWriter, data interface{}, resourceName string, errVal error) {
	if errVal != nil {
		webServerLog.Error(errVal, "Failed to list "+resourceName)
		http.Error(w, "Failed to list "+resourceName+": "+errVal.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		webServerLog.Error(err, "Failed to marshal "+resourceName+" to JSON")
		http.Error(w, "Failed to marshal "+resourceName+" to JSON: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, writeErr := w.Write(jsonData)
	if writeErr != nil {
		webServerLog.Error(writeErr, "Failed to write response for "+resourceName)
	}
}

// --- Cloud Provider Handlers ---
func (r *KubedeckReconciler) handleCloudClustersRequest(w http.ResponseWriter, req *http.Request) {
	provider := req.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "provider parameter is required", http.StatusBadRequest)
		return
	}

	var clusterProvider ClusterProvider
	switch provider {
	case "timeweb":
		clusterProvider = r.timewebProvider
	case "yandex":
		clusterProvider = r.yandexCloudProvider
	default:
		http.Error(w, "unsupported provider: "+provider, http.StatusBadRequest)
		return
	}

	clusters, err := clusterProvider.ListClusters(req.Context())
	writeJsonResponse(w, clusters, "cloud clusters", err)
}

func (r *KubedeckReconciler) handleCloudNodeGroupsRequest(w http.ResponseWriter, req *http.Request) {
	provider := req.URL.Query().Get("provider")
	clusterID := req.URL.Query().Get("cluster_id")

	if provider == "" || clusterID == "" {
		http.Error(w, "provider and cluster_id parameters are required", http.StatusBadRequest)
		return
	}

	var clusterProvider ClusterProvider
	switch provider {
	case "timeweb":
		clusterProvider = r.timewebProvider
	case "yandex":
		clusterProvider = r.yandexCloudProvider
	default:
		http.Error(w, "unsupported provider: "+provider, http.StatusBadRequest)
		return
	}

	nodeGroups, err := clusterProvider.GetNodeGroups(req.Context(), clusterID)
	writeJsonResponse(w, nodeGroups, "node groups", err)
}

func (r *KubedeckReconciler) handleCloudScaleNodeGroupRequest(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := req.URL.Query().Get("provider")
	clusterID := req.URL.Query().Get("cluster_id")
	groupID := req.URL.Query().Get("group_id")
	nodeCountStr := req.URL.Query().Get("node_count")

	if provider == "" || clusterID == "" || groupID == "" || nodeCountStr == "" {
		http.Error(w, "provider, cluster_id, group_id and node_count parameters are required", http.StatusBadRequest)
		return
	}

	nodeCount, err := strconv.Atoi(nodeCountStr)
	if err != nil {
		http.Error(w, "invalid node_count parameter", http.StatusBadRequest)
		return
	}

	var clusterProvider ClusterProvider
	switch provider {
	case "timeweb":
		clusterProvider = r.timewebProvider
	case "yandex":
		clusterProvider = r.yandexCloudProvider
	default:
		http.Error(w, "unsupported provider: "+provider, http.StatusBadRequest)
		return
	}

	err = clusterProvider.ScaleNodeGroup(req.Context(), clusterID, groupID, nodeCount)
	if err != nil {
		webServerLog.Error(err, "Failed to scale node group")
		http.Error(w, "Failed to scale node group: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":    true,
		"message":    "Node group scaled successfully",
		"provider":   provider,
		"cluster_id": clusterID,
		"group_id":   groupID,
		"node_count": nodeCount,
	}
	writeJsonResponse(w, response, "scale response", nil)
}

// --- Handlers for Kubernetes Resources ---
// startWebServer starts the HTTP server in a new goroutine.
func (r *KubedeckReconciler) startWebServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/pods", r.handlePodsRequest)                     //ok+
	mux.HandleFunc("/pv", r.handlePVRequest)                         //ok+
	mux.HandleFunc("/pvc", r.handlePVCRequest)                       //ok+
	mux.HandleFunc("/configmaps", r.handleConfigMapsRequest)         //ok+
	mux.HandleFunc("/services", r.handleServicesRequest)             //ok+
	mux.HandleFunc("/ingresses", r.handleIngressesRequest)           //ok+
	mux.HandleFunc("/deployments", r.handleDeploymentsRequest)       //ok+ ________+++++++++++++++
	mux.HandleFunc("/statefulsets", r.handleStatefulSetsRequest)     //ok+ ________
	mux.HandleFunc("/daemonsets", r.handleDaemonSetsRequest)         //ok+ ________
	mux.HandleFunc("/replicasets", r.handleReplicaSetsRequest)       //ok+ ________
	mux.HandleFunc("/namespaces", r.handleNamespacesRequest)         //ok+
	mux.HandleFunc("/nodes", r.handleNodesRequest)                   //ok+
	mux.HandleFunc("/storageclasses", r.handleStorageClassesRequest) //ok+
	mux.HandleFunc("/logs", r.handlePodLogsRequest)                  //ok+

	// Update для фронта
	mux.HandleFunc("/updatefromfront", r.HandleUpdateForFront)

	// Йобаный в рот, статистика блять cyka
	mux.HandleFunc("/analyze/resources", r.handleResourceAnalysisRequest)

	// Pods CRUD
	mux.HandleFunc("/pods/create", r.HandleCreatePod) //ok+
	mux.HandleFunc("/pods/update", r.HandleUpdatePod) //ok+
	mux.HandleFunc("/pods/delete", r.HandleDeletePod) //ok+

	// ConfigMaps CRUD
	mux.HandleFunc("/configmaps/create", r.HandleCreateConfigMap) //ok+
	mux.HandleFunc("/configmaps/update", r.HandleUpdateConfigMap)
	mux.HandleFunc("/configmaps/delete", r.HandleDeleteConfigMap)

	// Deployments CRUD
	mux.HandleFunc("/deployments/create", r.HandleCreateDeployment) //ok+
	mux.HandleFunc("/deployments/update", r.HandleUpdateDeployment) //ok+
	mux.HandleFunc("/deployments/delete", r.HandleDeleteDeployment)

	// Новые обработчики для метрик
	mux.HandleFunc("/metrics/cluster", r.handleClusterMetricsRequest)
	mux.HandleFunc("/metrics/pods", r.handlePodMetricsRequest)

	// PV CRUD
	mux.HandleFunc("/pv/create", r.HandleCreatePersistentVolume)
	mux.HandleFunc("/pv/update", r.HandleUpdatePersistentVolume)
	mux.HandleFunc("/pv/delete", r.HandleDeletePersistentVolume)

	// DaemonSets CRUD
	mux.HandleFunc("/daemonsets/create", r.HandleCreateDaemonSet)
	mux.HandleFunc("/daemonsets/update", r.HandleUpdateDaemonSet) //ok+
	mux.HandleFunc("/daemonsets/delete", r.HandleDeleteDaemonSet)

	// StatefulSet CRUD
	mux.HandleFunc("/statefulsets/create", r.HandleCreateStatefulSet)
	mux.HandleFunc("/statefulsets/update", r.HandleUpdateStatefulSet) //ok+
	mux.HandleFunc("/statefulsets/delete", r.HandleDeleteStatefulSet)

	// PVC CRUD
	mux.HandleFunc("/pvc/create", r.HandleCreatePersistentVolumeClaim)
	mux.HandleFunc("/pvc/update", r.HandleUpdatePersistentVolumeClaim)
	mux.HandleFunc("/pvc/delete", r.HandleDeletePersistentVolumeClaim)

	// Service CRUD
	mux.HandleFunc("/services/create", r.HandleCreateService)
	mux.HandleFunc("/services/update", r.HandleUpdateService)
	mux.HandleFunc("/services/delete", r.HandleDeleteService)

	// Ingress CRUD
	mux.HandleFunc("/ingresses/create", r.HandleCreateIngress)
	mux.HandleFunc("/ingresses/update", r.HandleUpdateIngress)
	mux.HandleFunc("/ingresses/delete", r.HandleDeleteIngress)

	// StorageClass CRUD
	mux.HandleFunc("/storageclasses/create", r.HandleCreateStorageClass)
	mux.HandleFunc("/storageclasses/update", r.HandleUpdateStorageClass)
	mux.HandleFunc("/storageclasses/delete", r.HandleDeleteStorageClass)

	// Node CRUD
	mux.HandleFunc("/nodes/update", r.HandleUpdateNode)
	mux.HandleFunc("/nodes/delete", r.HandleDeleteNode)

	// ReplicaSet CRUD
	mux.HandleFunc("/replicasets/create", r.HandleCreateReplicaSet)
	mux.HandleFunc("/replicasets/update", r.HandleUpdateReplicaSet) //ok хз он не делает replicas их readi
	mux.HandleFunc("/replicasets/delete", r.HandleDeleteReplicaSet)

	// Namespaces CRUD (пример для кластерного)
	mux.HandleFunc("/namespaces/create", r.HandleCreateNamespace)

	// Cloud Provider Handlers
	mux.HandleFunc("/cloud/clusters", r.handleCloudClustersRequest)
	mux.HandleFunc("/cloud/nodegroups", r.handleCloudNodeGroupsRequest)
	mux.HandleFunc("/cloud/scale", r.handleCloudScaleNodeGroupRequest)

	mux.HandleFunc("/telegram/config", r.handleTelegramBotConfigRequest)

	webServerLog.Info("Starting web server", "port", 8999)
	if err := http.ListenAndServe(":8999", mux); err != nil {
		webServerLog.Error(err, "Failed to start web server")
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *KubedeckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize cloud providers
	timewebProvider, err := NewClusterProvider("timeweb")
	if err != nil {
		webServerLog.Error(err, "Failed to initialize TimeWeb provider")
	} else {
		r.timewebProvider = timewebProvider
	}

	yandexProvider, err := NewClusterProvider("yandex")
	if err != nil {
		webServerLog.Error(err, "Failed to initialize Yandex Cloud provider")
	} else {
		r.yandexCloudProvider = yandexProvider
	}

	// Initialize Telegram bot settings
	r.TelegramBotSettings = NewTelegramBotSettings()

	// Start the web server in a goroutine.
	go r.startWebServer()

	// Start Telegram bot in a goroutine
	go r.StartTelegramBot(context.Background())

	return ctrl.NewControllerManagedBy(mgr).
		For(&ctrlv1.Kubedeck{}). // Assuming Kubedeck is your primary CRD
		Named("kubedeck").
		Complete(r)
}
