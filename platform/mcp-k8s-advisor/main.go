package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Estructuras para el protocolo JSON-RPC de MCP
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// obtenerKubeConfig detecta dinámicamente el entorno (In-Cluster o Local via KUBECONFIG)
func obtenerKubeConfig() (*rest.Config, error) {
	// 1. Intentar con la configuración interna del Pod (Producción en K8s)
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// 2. Si falla, estamos local (Desarrollo con túnel SSH)
	fmt.Fprintln(os.Stderr, "ℹ️ Fuera del clúster. Buscando configuración local...")
	kubeconfigEnv := os.Getenv("KUBECONFIG")

	if kubeconfigEnv == "" {
		home, _ := os.UserHomeDir()
		kubeconfigEnv = filepath.Join(home, ".kube", "config")
	}

	return clientcmd.BuildConfigFromFlags("", kubeconfigEnv)
}

func main() {
	fmt.Fprintln(os.Stderr, "🤖 Servidor MCP Kubernetes Advisor Iniciado...")

	// Inicializar cliente de Kubernetes
	cfg, err := obtenerKubeConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al obtener kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al crear clientset: %v\n", err)
		os.Exit(1)
	}

	// Canal para atrapar las señales de terminación del sistema (SIGINT, SIGTERM)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Levantamos el loop de escucha de la entrada estándar (STDIN) en una Goroutine
	// Para que el fin de archivo (EOF) no mate el proceso principal de inmediato
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			var req Request
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}

			var resp Response
			resp.JSONRPC = "2.0"
			resp.ID = req.ID

			// Manejo de métodos del protocolo MCP
			switch req.Method {
			case "tools/list":
				resp.Result = ManejarToolsList()

			case "tools/call":
				resp.Result, err = ManejarToolsCall(req.Params, clientset)
				if err != nil {
					resp.Error = map[string]string{"message": err.Error()}
				}

			default:
				resp.Error = map[string]string{"message": "Método no soportado"}
			}

			// Responder por la salida estándar (STDOUT)
			jsonResp, _ := json.Marshal(resp)
			fmt.Println(string(jsonResp))
		}
	}()

	// El hilo principal se bloquea acá, consumiendo 0% CPU, hasta que K8s mande una señal
	fmt.Fprintln(os.Stderr, "📌 Servidor bloqueado esperando señales del clúster (SIGTERM/SIGINT)...")
	<-sigs

	fmt.Fprintln(os.Stderr, "👋 Señal recibida. Cerrando el servidor limpiamente...")
}
