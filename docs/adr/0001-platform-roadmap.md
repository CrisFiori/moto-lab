# ADR 0001: Roadmap de Evolución de la Plataforma Moto-Lab

## Estatus
Aceptado — actualizado tras migración a infraestructura bare-metal (ver ADR 0002)

## Contexto
Para transicionar de forma sólida hacia roles de Platform Engineering, se requiere el
diseño de una plataforma interna (IDP) que resuelva abstracciones de red, observabilidad,
seguridad y automatización sobre un clúster local. Este documento define el diseño
evolutivo del laboratorio.

> **Nota de contexto (actualizada):** el diseño original asumía Kind sobre VirtualBox
> como sustrato. Un incidente de degradación de red (NAT de VirtualBox conviviendo con
> Hyper-V) motivó la migración a **k3s sobre Ubuntu Server bare-metal (dual boot)**.
> El detalle técnico de esa migración —Traefik reemplazando Envoy Gateway, ServiceLB
> reemplazando MetalLB, adopción temprana de GHCR— está registrado en ADR 0002. Las
> fases de este roadmap siguen vigentes conceptualmente; donde el sustrato cambia el
> *cómo* pero no el *qué*, se anota abajo.

## Estado de Progreso (actualizado)

| Fase | Estado | Nota |
|---|---|---|
| Fase 1 — Fundamentos GitOps | 🟡 Parcial | App of Apps operativo sobre k3s bare-metal. **Sealed Secrets aún no implementado** — pendiente real, no asumir hecho. |
| Fase 2 — Observabilidad | ⚪ No iniciada | — |
| Fase 3 — Seguridad como Código | ⚪ No iniciada | — |
| Fase 4 — CI/CD Pipeline Real | 🟡 Adelantado parcialmente | Registro real (GHCR) y **tags inmutables** ya adoptados como consecuencia de la migración de infra, antes de lo planeado. Falta: pipeline automatizado en GitHub Actions, Staging/Producción por namespace. |
| Fase 5 — Backstage IDP | ⚪ No iniciada | — |
| Fase 6 — Portfolio y Cierre | ⚪ No iniciada | — |

## Decisión de Arquitectura: Plan de 6 Fases

### Fase 1 — Fundamentos GitOps (Semana 1)
* **Técnico:** Sincronización declarativa con ArgoCD sobre **k3s bare-metal**. Estructura
  basada en el patrón *App of Apps*. Integración de Sealed Secrets para el manejo seguro
  de secretos en repositorios públicos *(pendiente — actualmente la gestión de
  credenciales sensibles, como el password de ArgoCD, es manual vía `kubectl patch`
  sobre `Secret`, no versionada en Git)*. Despliegue de muestra automatizado.
* **Conceptual:** Entender el modelo Pull vs Push de GitOps y control de estado deseado.
  Lección adicional no planeada: un `include` selectivo en el root Application es
  potente pero silencioso — un archivo puente faltante deja partes enteras del sistema
  fuera de gestión sin generar error visible. Ver ADR 0002.

### Fase 2 — Observabilidad como Plataforma (Semana 2-3)
* **Técnico:** Stack de telemetría con `kube-prometheus-stack` (Prometheus + Grafana) y
  Loki/Promtail para logs. Creación de dashboards y alertas de plataforma (Pod
  CrashLoopBackOff, alta utilización).
* **Conceptual:** Definición práctica de SLIs, SLOs y SLAs. Introducción a métricas DORA.

### Fase 3 — Seguridad como Código (Semana 4-5)
* **Técnico:** Políticas de gobernanza con Kyverno (restricción de límites de recursos,
  labels obligatorios, bloqueo de mutaciones inseguras). Escaneo continuo de imágenes
  de contenedores mediante Trivy Operator.
* **Conceptual:** Disciplina de Policy as Code y cultura de *Shift-Left Security*.

### Fase 4 — CI/CD Pipeline Real (Semana 6-7)
* **Técnico:** Pipeline completo en GitHub Actions: compilación, testeo, construcción y
  subida de imágenes **a GHCR (ya adoptado, ver ADR 0002)**, actualización dinámica del
  manifiesto en Git y trigger automático en ArgoCD. Manejo de Staging y Producción vía
  namespaces. Reemplazar el flujo manual actual (`docker build` + `docker push` a mano)
  por automatización real en Actions.
* **Conceptual:** Trunk-Based Development vs GitFlow.

### Fase 5 — Backstage IDP - El Core (Semana 8-10)
* **Técnico:** Despliegue de Spotify Backstage vía Helm. Integración de plugins nativos
  de ArgoCD y Kubernetes. Creación de un *Software Template* para automatizar el
  scaffolding de un microservicio desde el portal.
* **Conceptual:** Dinámicas organizacionales de *Team Topologies* y abstracción de la
  experiencia de desarrollo (DevX).

### Fase 6 — Portfolio y Cierre (Semana 11)
* **Técnico:** Scripting de bootstrap global del laboratorio y empaquetado de lecciones
  aprendidas.
* **Conceptual:** Simulación de auditorías de arquitectura enfocadas en el ecosistema
  Fintech.

## Consecuencias
Cada herramienta que se agregue al clúster a partir de este punto deberá responder y
alinearse cronológicamente con las fases descritas en este registro, manteniendo la
consistencia de la plataforma. Los cambios de sustrato de infraestructura (bare-metal,
Traefik, GHCR) se documentan en ADRs separados y numerados, sin reescribir la historia
de este documento — este ADR describe la intención original; los ADRs subsiguientes
documentan la realidad y sus desvíos justificados.