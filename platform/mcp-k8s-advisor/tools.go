package main

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ListarNodos consulta la API de Kubernetes y devuelve un resumen del estado de los nodos.
func ListarNodos(clientset *kubernetes.Clientset) (string, error) {
	// Consúltale al clúster de Kind la lista de nodos reales
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("error al listar nodos: %v", err)
	}

	resultado := "📊 [MCP Advisor] Estado actual de los nodos en el clúster:\n"
	for _, node := range nodes.Items {
		status := "NotReady"
		// Buscamos la condición de "Ready" para saber si el nodo está operativo
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" && cond.Status == "True" {
				status = "Ready"
			}
		}
		resultado += fmt.Sprintf("  • Nodo: %s | Status: %s\n", node.Name, status)
	}

	return resultado, nil
}
