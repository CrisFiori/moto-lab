# ADR 0001: Roadmap de Evolución de la Plataforma Moto-Lab

## Estatus
Aceptado

## Contexto
Para transicionar de forma sólida hacia roles de Platform Engineering, se requiere el diseño de una plataforma interna (IDP) que resuelva abstracciones de red, observabilidad, seguridad y automatización sobre un clúster local (Kind). Este documento define el diseño evolutivo del laboratorio.

## Decisión de Arquitectura: Plan de 6 Fases

### Fase 1 — Fundamentos GitOps (Semana 1)
* **Técnico:** Sincronización declarativa con ArgoCD. Estructura basada en el patrón *App of Apps*. Integración de Sealed Secrets para el manejo seguro de secretos en repositorios públicos. Despliegue de muestra automatizado.
* **Conceptual:** Entender el modelo Pull vs Push de GitOps y control de estado deseado.

### Fase 2 — Observabilidad como Plataforma (Semana 2-3)
* **Técnico:** Stack de telemetría con `kube-prometheus-stack` (Prometheus + Grafana) y Loki/Promtail para logs. Creación de dashboards y alertas de plataforma (Pod CrashLoopBackOff, alta utilización).
* **Conceptual:** Definición práctica de SLIs, SLOs y SLAs. Introducción a métricas DORA.

### Fase 3 — Seguridad como Código (Semana 4-5)
* **Técnico:** Políticas de gobernanza con Kyverno (restricción de límites de recursos, labels obligatorios, bloqueo de mutaciones inseguras). Escaneo continuo de imágenes de contenedores mediante Trivy Operator.
* **Conceptual:** Disciplina de Policy as Code y cultura de *Shift-Left Security*.

### Fase 4 — CI/CD Pipeline Real (Semana 6-7)
* **Técnico:** Pipeline completo en GitHub Actions: compilación, testeo, construcción y subida de imágenes, actualización dinámica del manifiesto en Git y trigger automático en ArgoCD. Manejo de Staging y Producción vía namespaces.
* **Conceptual:** Trunk-Based Development vs GitFlow.

### Fase 5 — Backstage IDP - El Core (Semana 8-10)
* **Técnico:** Despliegue de Spotify Backstage vía Helm. Integración de plugins nativos de ArgoCD y Kubernetes. Creación de un *Software Template* para automatizar el scaffolding de un microservicio desde el portal.
* **Conceptual:** Dinámicas organizacionales de *Team Topologies* y abstracción de la experiencia de desarrollo (DevX).

### Fase 6 — Portfolio y Cierre (Semana 11)
* **Técnico:** Scripting de bootstrap global del laboratorio y empaquetado de lecciones aprendidas.
* **Conceptual:** Simulación de auditorías de arquitectura enfocadas en el ecosistema Fintech.

## Consecuencias
Cada herramienta que se agregue al clúster a partir de este punto deberá responder y alinearse cronológicamente con las fases descritas en este registro, manteniendo la consistencia de la plataforma.

