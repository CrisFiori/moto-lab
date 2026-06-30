package main

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Estructuras de respuesta para cumplir con la especificación MCP
type ToolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolCallResult struct {
	Content []TextContent `json:"content"`
}

type CallArgs struct {
	Name string `json:"name"`
}

// ManejarToolsList expone las herramientas que el Advisor sabe usar
func ManejarToolsList() ToolsListResult {
	return ToolsListResult{
		Tools: []ToolDefinition{
			{
				Name:        "get_cluster_nodes",
				Description: "Obtiene el listado de nodos del clúster de Kubernetes y su estado de salud actual.",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
	}
}

// ManejarToolsCall procesa la ejecución de la herramienta solicitada
func ManejarToolsCall(paramsRaw json.RawMessage, clientset *kubernetes.Clientset) (ToolCallResult, error) {
	var args CallArgs
	if err := json.Unmarshal(paramsRaw, &args); err != nil {
		return ToolCallResult{}, err
	}

	switch args.Name {
	case "get_cluster_nodes":
		return ListarNodos(clientset)
	default:
		return ToolCallResult{}, fmt.Errorf("herramienta no encontrada: %s", args.Name)
	}
}

// ListarNodos interactúa directamente con la API de Kubernetes
func ListarNodos(clientset *kubernetes.Clientset) (ToolCallResult, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return ToolCallResult{}, fmt.Errorf("error al listar nodos: %v", err)
	}

	resultadoTexto := "=== Nodos del Clúster Kubernetes ===\n"
	for _, node := range nodes.Items {
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" {
				status = string(cond.Status)
				break
			}
		}
		resultadoTexto += fmt.Sprintf("- Nodo: %s | Estado Ready: %s | Versión Kubelet: %s\n",
			node.Name, status, node.Status.NodeInfo.KubeletVersion)
	}

	return ToolCallResult{
		Content: []TextContent{
			{
				Type: "text",
				Text: resultadoTexto,
			},
		},
	}, nil
}
