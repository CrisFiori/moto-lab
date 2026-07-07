# ADR 0002: Migración a Infraestructura Bare-Metal (k3s) y Rediseño de Networking

## Estatus
Aceptado

## Contexto
El laboratorio operaba originalmente sobre Kind, corriendo dentro de una VM de
VirtualBox (Ubuntu Server) sobre un host Windows con Hyper-V habilitado. Durante el
desarrollo de Fase 1, se detectó una degradación severa de throughput TCP (~7 KB/s)
en la comunicación del ArgoCD repo-server, causada por la coexistencia de la NAT de
VirtualBox con Hyper-V a nivel de host. Se aplicaron mitigaciones parciales (deshabilitar
NIC offloading, ajustar MTU a 1400 vía netplan), pero el problema de fondo era estructural
al stack de virtualización elegido, no resoluble de forma definitiva sin cambiar el
sustrato.

Se decidió instalar Ubuntu Server en dual boot directo sobre el hardware físico
(Dell Latitude, 32GB RAM, i7-1255U), eliminando la capa de virtualización de red por
completo.

## Decisión de Arquitectura

### 1. Kind → k3s
Kind (Kubernetes-in-Docker) fue reemplazado por k3s instalado directo sobre el SO bare-metal.
Justificación: k3s está diseñado para topologías single-node/edge, no requiere Docker como
dependencia intermedia, y expone directamente los mecanismos de red del host sin capas
adicionales de NAT.

### 2. MetalLB → ServiceLB (Klipper)
En un entorno virtualizado sin LoadBalancer real, MetalLB era necesario para asignar IPs a
Services tipo `LoadBalancer`. En bare-metal, k3s incluye ServiceLB (Klipper LB) de fábrica,
que bindea directamente sobre la IP del nodo. Se eliminó la dependencia externa.

### 3. Envoy Gateway (Gateway API) → Traefik (Ingress clásico vía IngressRoute/Middleware)
Se evaluaron tres alternativas para networking norte-sur:

| Opción | Pros | Contras |
|---|---|---|
| Envoy Gateway + Gateway API | Estándar más moderno, dirección futura del ecosistema | CRDs adicionales a mantener, mayor superficie de troubleshooting en un momento de estabilización crítica |
| Traefik + Gateway API | Balance entre estándar moderno y motor probado | Requiere instalar CRDs de Gateway API por fuera de k3s + habilitar provider experimental vía HelmChartConfig — fricción no justificada para el objetivo del día |
| **Traefik + Ingress clásico (IngressRoute/Middleware)** ✅ | Viene integrado en k3s sin instalación adicional, CRDs propias listas de fábrica, middlewares nativos potentes (RateLimit, IPAllowList, Headers, ForwardAuth) para seguridad perimetral | Ingress clásico está en modo mantenimiento, no evolutivo; migración futura a Gateway API queda como deuda técnica consciente |

Se optó por la tercera opción. La prioridad del día era estabilizar el cluster completo
(ArgoCD, GitOps, apps) en tiempo acotado; Gateway API queda evaluado y descartado
*por ahora*, no por desconocimiento sino por relación costo/beneficio en este momento
del proyecto.

### 4. Imágenes: import manual → GHCR con tags inmutables
Kind permitía cargar imágenes directo al containerd del cluster (`kind load docker-image`),
sin necesidad de un registry real. k3s bare-metal no tiene ese atajo. Se evaluaron tres
opciones (import manual a containerd, registry local en cluster, GHCR) y se adoptó
**GHCR (GitHub Container Registry)** adelantando una decisión ya planeada para la Fase 4
del roadmap (ADR 0001), en vez de resolverlo de forma transitoria. Se estableció además
la convención de **tags inmutables** (`vX.Y.Z`, no `:latest`), ya identificada como
antipattern a evitar en el roadmap original — esto garantiza que ArgoCD detecte cambios
reales de versión en cada sync en lugar de asumir un estado desactualizado.

### 5. Gestión de cuentas ArgoCD
Se recreó la cuenta nominal `cfiori` de forma declarativa (`ConfigMap` `argocd-cm` +
`argocd-rbac-cm` versionados en Git) y se deshabilitó la cuenta `admin` una vez
verificado el acceso. El hash bcrypt del password se aplicó manualmente sobre el
`Secret` `argocd-secret` en cluster — **no versionado en Git**, marcado como deuda
técnica pendiente para Sealed Secrets (Fase 1 de ADR 0001, aún no implementada).

### 6. Topología de nodo único
Se mantiene control-plane + worker en el mismo nodo. k3s no aplica taint de
`NoSchedule` al nodo server por diseño, lo cual es coherente con un lab de nodo único.
Separación de roles (taints/tolerations) queda como ejercicio de portfolio futuro,
no como necesidad actual — HA de etcd/control-plane no es una preocupación relevante
en un entorno de un solo nodo físico.

## Consecuencias

**Positivas:**
- Eliminación completa del problema de red de origen (NAT VirtualBox/Hyper-V).
- Stack más liviano: menos CRDs, menos componentes a mantener (sin MetalLB, sin Envoy Gateway).
- Registro de imágenes real y versionado, adelantando trabajo de Fase 4.
- Gestión de cuentas ArgoCD declarativa, alineada con higiene de credenciales esperable
  en entornos regulados.
- Se detectaron y corrigieron dos gaps de gobernanza GitOps preexistentes (Applications
  `business-apps` y `platform-apps` referenciadas en `root-app.yaml` pero sin archivo
  puente correspondiente, dejando esas cargas fuera de control declarativo sin error
  visible) — hallazgo no planeado, documentado como lección de proceso.

**Negativas / deuda técnica consciente:**
- Se abandona Gateway API como estándar activo en el cluster; requiere migración
  explícita futura si el objetivo pasa a incluir demostración de networking moderno.
- Sealed Secrets sigue sin implementarse; el password de ArgoCD y futuros secrets
  quedan gestionados manualmente fuera de Git.
- Topología single-node no ejercita separación real de roles de cluster (taints/tolerations),
  relevante para escenarios productivos multi-nodo tipo EKS.
- El pipeline de build/push a GHCR es manual (`docker build` + `docker push` local);
  la automatización vía GitHub Actions sigue pendiente como parte de Fase 4.