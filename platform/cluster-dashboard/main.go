package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClusterData representa el JSON que le vamos a mandar al Frontend
type ClusterData struct {
	K8sVersion string     `json:"k8sVersion"`
	TotalNodes int        `json:"totalNodes"`
	Nodes      []NodeInfo `json:"nodes"`
}

type NodeInfo struct {
	Name             string `json:"name"`
	Status           string `json:"status"`
	Role             string `json:"role"`
	CPUCapacity      string `json:"cpu"`
	MemoryCapacity   string `json:"memory"`
	ContainerRuntime string `json:"containerRuntime"`
}

// AppInfo representa el JSON de salida del módulo de ArgoCD Applications
type AppInfo struct {
	Name   string `json:"name"`
	Sync   string `json:"sync"`   // "Synced" / "OutOfSync"
	Health string `json:"health"` // "Healthy" / "Progressing" / "Degraded"
}

// GVR (GroupVersionResource) del CRD Application de ArgoCD.
// Se arma a partir del apiVersion "argoproj.io/v1alpha1" que usás en tus YAML de ArgoCD.
var argoAppGVR = schema.GroupVersionResource{
	Group:    "argoproj.io",
	Version:  "v1alpha1",
	Resource: "applications",
}

// Lógica de detección de entorno para obtener el Kubeconfig (Igual al MCP)
func obtenerKubeConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if kubeconfigEnv == "" {
		home, _ := os.UserHomeDir()
		kubeconfigEnv = filepath.Join(home, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
}

func main() {
	fmt.Println("🚀 Iniciando el Cluster Dashboard Backend...")

	// 1. Inicializamos la conexión con el API Server
	cfg, err := obtenerKubeConfig()
	if err != nil {
		fmt.Printf("❌ Error de configuración: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("❌ Error al crear clientset: %v\n", err)
		os.Exit(1)
	}

	// Dynamic client: mismo config (cfg), pero para recursos no nativos (CRDs como Application)
	dynClient, err := dynamic.NewForConfig(cfg)
	if err != nil {
		fmt.Printf("❌ Error al crear dynamic client: %v\n", err)
		os.Exit(1)
	}

	// 2. Mapeamos los Handlers (Endpoints) de nuestro servidor web

	// Endpoint de nodos (ya existente)
	http.HandleFunc("/api/v1/cluster-info", srcClusterInfoHandler(clientset))

	// Endpoint nuevo: estado de las Applications de ArgoCD
	http.HandleFunc("/api/v1/argo-apps", srcArgoAppsHandler(dynClient))

	// Endpoint de salud para los Probes de Kubernetes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Servidor de archivos estáticos para el Frontend (HTML/JS/CSS)
	fs := http.FileServer(http.Dir("./ui"))
	http.Handle("/", fs)

	// 3. Arrancamos el loop del servidor en el puerto 8080 (Función bloqueante)
	fmt.Println("🌍 Servidor escuchando en http://0.0.0.0:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("❌ Error al levantar el servidor: %v\n", err)
		os.Exit(1)
	}
}

// srcClusterInfoHandler devuelve info de versión y nodos del cluster
func srcClusterInfoHandler(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ctx := context.Background()

		versionInfo, err := clientset.Discovery().ServerVersion()
		k8sVersion := "Desconocida"
		if err == nil {
			k8sVersion = versionInfo.GitVersion
		}

		nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "No se pudieron listar los nodos: %v"}`, err), http.StatusInternalServerError)
			return
		}

		var nodosInfo []NodeInfo
		for _, node := range nodeList.Items {
			status := "NotReady"
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					status = "Ready"
					break
				}
			}

			role := "worker"
			if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
				role = "control-plane"
			}

			nodosInfo = append(nodosInfo, NodeInfo{
				Name:             node.Name,
				Status:           status,
				Role:             role,
				CPUCapacity:      node.Status.Capacity.Cpu().String(),
				MemoryCapacity:   node.Status.Capacity.Memory().String(),
				ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			})
		}

		data := ClusterData{
			K8sVersion: k8sVersion,
			TotalNodes: len(nodeList.Items),
			Nodes:      nodosInfo,
		}

		json.NewEncoder(w).Encode(data)
	}
}

// srcArgoAppsHandler devuelve el estado de sync/health de las Applications de ArgoCD
func srcArgoAppsHandler(dynClient dynamic.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		ctx := context.Background()

		list, err := dynClient.Resource(argoAppGVR).
			Namespace("argocd").
			List(ctx, metav1.ListOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "No se pudieron listar las Applications: %v"}`, err), http.StatusInternalServerError)
			return
		}

		var apps []AppInfo
		for _, item := range list.Items {
			syncStatus, _, _ := unstructured.NestedString(item.Object, "status", "sync", "status")
			healthStatus, _, _ := unstructured.NestedString(item.Object, "status", "health", "status")

			apps = append(apps, AppInfo{
				Name:   item.GetName(),
				Sync:   syncStatus,
				Health: healthStatus,
			})
		}

		json.NewEncoder(w).Encode(apps)
	}
}
