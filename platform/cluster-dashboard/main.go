package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// 2. Mapeamos los Handlers (Endpoints) de nuestro servidor web

	// Endpoint de la API que consume el Front
	http.HandleFunc("/api/v1/cluster-info", srcClusterInfoHandler(clientset))

	// Endpoint de salud para los Probes de Kubernetes
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Servidor de archivos estáticos para el Frontend (HTML/JS/CSS)
	// Va a buscar los archivos dentro de una carpeta llamada 'ui' al mismo nivel del binario
	fs := http.FileServer(http.Dir("./ui"))
	http.Handle("/", fs)

	// 3. Arrancamos el loop del servidor en el puerto 8080 (Función bloqueante)
	fmt.Println("🌍 Servidor escuchando en http://0.0.0.0:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("❌ Error al levantar el servidor: %v\n", err)
		os.Exit(1)
	}
}

// srcClusterInfoHandler es una función de alto orden (devuelve un Handler HTTP)
// Le pasamos el 'clientset' para que la función interna pueda consultar a Kubernetes
func srcClusterInfoHandler(clientset *kubernetes.Clientset) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Seteamos los headers para manejar CORS y decirle al navegador que mandamos JSON
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// context.Background() es obligatorio en llamadas modernas de client-go para manejar timeouts
		ctx := context.Background()

		// A. Consultar la versión del clúster
		versionInfo, err := clientset.Discovery().ServerVersion()
		k8sVersion := "Desconocida"
		if err == nil {
			k8sVersion = versionInfo.GitVersion
		}

		// B. Consultar los Nodos del clúster (Llamada Cluster-Wide)
		// metav1.ListOptions{} vacío significa: "Traeme todos los nodos sin filtros ni labels"
		nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "No se pudieron listar los nodos: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// C. Procesar y formatear la información de los nodos
		var nodosInfo []NodeInfo
		for _, node := range nodeList.Items {
			// Evaluamos el estado (Status) del nodo recorriendo sus condiciones
			status := "NotReady"
			for _, cond := range node.Status.Conditions {
				if cond.Type == "Ready" && cond.Status == "True" {
					status = "Ready"
					break
				}
			}

			// Identificamos el rol del nodo buscando en sus labels nativos
			role := "worker"
			if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
				role = "control-plane"
			}

			// Armamos nuestra estructura limpia
			nodosInfo = append(nodosInfo, NodeInfo{
				Name:             node.Name,
				Status:           status,
				Role:             role,
				CPUCapacity:      node.Status.Capacity.Cpu().String(),
				MemoryCapacity:   node.Status.Capacity.Memory().String(),
				ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
			})
		}

		// D. Empaquetamos todo en la estructura final
		data := ClusterData{
			K8sVersion: k8sVersion,
			TotalNodes: len(nodeList.Items),
			Nodes:      nodosInfo,
		}

		// E. Serializamos a JSON y respondemos al cliente
		json.NewEncoder(w).Encode(data)
	}
}
