package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Estructura para decodificar lo que nos manda la IA (JSON-RPC básico)
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"`
}

// Estructura para armar la respuesta que espera la IA
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// Formato específico del protocolo MCP para devolver texto a un LLM
type MCPResult struct {
	Content []MCPContent `json:"content"`
}

type MCPContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func main() {
	fmt.Fprintln(os.Stderr, "🤖 Servidor MCP Kubernetes Advisor Iniciado...")

	// 1. Conexión nativa al clúster (InClusterConfig)
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error de InClusterConfig (Solo funciona dentro de K8s): %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error al crear el cliente de K8s: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "✅ Conexión con la API de Kubernetes establecida.")

	// 2. Loop de escucha del Protocolo MCP (stdin -> stdout)
	// Usamos un scanner para leer línea por línea lo que la IA nos mande
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req JSONRPCRequest
		linea := scanner.Bytes()

		// Decodificamos el JSON que mandó la IA
		if err := json.Unmarshal(linea, &req); err != nil {
			fmt.Fprintf(os.Stderr, "⚠️ Error al decodificar JSON-RPC: %v\n", err)
			continue
		}

		// Inicializamos la respuesta estándar
		resp := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		// 3. Ruteo de Métodos del Estándar MCP
		switch req.Method {
		case "tools/list":
			// Le contamos a Jarvis qué herramientas tenemos disponibles
			resp.Result = map[string]interface{}{
				"tools": []map[string]interface{}{
					{
						"name":        "get_cluster_nodes",
						"description": "Lista los nodos del clúster de Kubernetes y muestra su estado de salud actual (Ready/NotReady).",
						"inputSchema": map[string]interface{}{
							"type":       "object",
							"properties": map[string]interface{}{},
						},
					},
				},
			}

		case "tools/call":
			// Jarvis decidió ejecutar una herramienta. Ejecutamos nuestra función de tools.go
			textoResultado, err := ListarNodos(clientset)
			if err != nil {
				resp.Error = map[string]string{"message": err.Error()}
			} else {
				resp.Result = MCPResult{
					Content: []MCPContent{
						{Type: "text", Text: textoResultado},
					},
				}
			}

		default:
			resp.Error = map[string]string{"message": "Método no soportado por este servidor MCP"}
		}

		// 4. Enviamos la respuesta de vuelta a la IA por stdout
		jsonResp, _ := json.Marshal(resp)
		fmt.Println(string(jsonResp))
	}
}
