# Guide de Déploiement Kubernetes

> **Version** : 1.0.0  
> **Dernière mise à jour** : Novembre 2025

## Prérequis

- Cluster Kubernetes 1.25+
- kubectl configuré
- Helm 3+ (optionnel)
- Image Docker disponible dans un registre accessible

## Déploiement Rapide

### Manifest Minimal

```yaml
# fibcalc-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fibcalc
  labels:
    app: fibcalc
spec:
  replicas: 2
  selector:
    matchLabels:
      app: fibcalc
  template:
    metadata:
      labels:
        app: fibcalc
    spec:
      containers:
        - name: fibcalc
          image: ghcr.io/your-org/fibcalc:latest
          args: ["--server", "--port", "8080"]
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "2000m"
              memory: "2Gi"
---
apiVersion: v1
kind: Service
metadata:
  name: fibcalc
spec:
  selector:
    app: fibcalc
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
```

```bash
kubectl apply -f fibcalc-deployment.yaml
```

## Déploiement Complet

### Namespace

```yaml
# namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: fibcalc
  labels:
    name: fibcalc
```

### ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: fibcalc-config
  namespace: fibcalc
data:
  THRESHOLD: "4096"
  FFT_THRESHOLD: "1000000"
  STRASSEN_THRESHOLD: "3072"
  TIMEOUT: "5m"
```

### Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fibcalc
  namespace: fibcalc
  labels:
    app: fibcalc
    version: "1.0.0"
spec:
  replicas: 3
  selector:
    matchLabels:
      app: fibcalc
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: fibcalc
        version: "1.0.0"
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        fsGroup: 1000
      containers:
        - name: fibcalc
          image: ghcr.io/your-org/fibcalc:1.0.0
          imagePullPolicy: IfNotPresent
          args:
            - "--server"
            - "--port"
            - "8080"
            - "--auto-calibrate"
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          env:
            - name: GOMAXPROCS
              valueFrom:
                resourceFieldRef:
                  resource: limits.cpu
          envFrom:
            - configMapRef:
                name: fibcalc-config
          resources:
            requests:
              cpu: "500m"
              memory: "512Mi"
            limits:
              cpu: "2000m"
              memory: "2Gi"
          livenessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /health
              port: http
            initialDelaySeconds: 5
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 2
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
```

### Service

```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: fibcalc
  namespace: fibcalc
  labels:
    app: fibcalc
spec:
  type: ClusterIP
  selector:
    app: fibcalc
  ports:
    - name: http
      port: 80
      targetPort: http
      protocol: TCP
```

### Ingress (avec nginx-ingress)

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: fibcalc
  namespace: fibcalc
  annotations:
    kubernetes.io/ingress.class: nginx
    nginx.ingress.kubernetes.io/rate-limit: "100"
    nginx.ingress.kubernetes.io/rate-limit-window: "1m"
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - api.fibonacci.example.com
      secretName: fibcalc-tls
  rules:
    - host: api.fibonacci.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: fibcalc
                port:
                  number: 80
```

### HorizontalPodAutoscaler

```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: fibcalc
  namespace: fibcalc
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: fibcalc
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 4
          periodSeconds: 15
      selectPolicy: Max
```

### PodDisruptionBudget

```yaml
# pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: fibcalc
  namespace: fibcalc
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: fibcalc
```

### NetworkPolicy

```yaml
# networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: fibcalc
  namespace: fibcalc
spec:
  podSelector:
    matchLabels:
      app: fibcalc
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
        - namespaceSelector:
            matchLabels:
              name: monitoring
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: UDP
          port: 53  # DNS
```

## Déploiement

### Application des Manifests

```bash
# Créer le namespace
kubectl apply -f namespace.yaml

# Appliquer les configurations
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f pdb.yaml
kubectl apply -f networkpolicy.yaml

# Optionnel: Ingress
kubectl apply -f ingress.yaml
```

### Vérification

```bash
# État des pods
kubectl get pods -n fibcalc

# Logs
kubectl logs -f deployment/fibcalc -n fibcalc

# Port-forward pour test local
kubectl port-forward svc/fibcalc 8080:80 -n fibcalc

# Test
curl http://localhost:8080/health
```

## Helm Chart (Optionnel)

### Structure

```
charts/fibcalc/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── hpa.yaml
│   ├── configmap.yaml
│   └── ingress.yaml
```

### Chart.yaml

```yaml
apiVersion: v2
name: fibcalc
description: High-performance Fibonacci calculator
type: application
version: 1.0.0
appVersion: "1.0.0"
```

### values.yaml

```yaml
replicaCount: 2

image:
  repository: ghcr.io/your-org/fibcalc
  tag: "1.0.0"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 80

ingress:
  enabled: false
  className: nginx
  hosts:
    - host: api.fibonacci.example.com
      paths:
        - path: /
          pathType: Prefix

resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

config:
  threshold: 4096
  fftThreshold: 1000000
  strassenThreshold: 3072
  timeout: 5m
```

### Installation

```bash
# Installation
helm install fibcalc ./charts/fibcalc -n fibcalc --create-namespace

# Mise à jour
helm upgrade fibcalc ./charts/fibcalc -n fibcalc

# Avec valeurs personnalisées
helm install fibcalc ./charts/fibcalc \
  -n fibcalc \
  --set replicaCount=5 \
  --set resources.limits.cpu=4000m \
  --set ingress.enabled=true
```

## Monitoring

### ServiceMonitor (Prometheus Operator)

```yaml
# servicemonitor.yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: fibcalc
  namespace: fibcalc
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: fibcalc
  endpoints:
    - port: http
      path: /metrics
      interval: 15s
```

### Alertes

```yaml
# prometheusrule.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: fibcalc
  namespace: fibcalc
spec:
  groups:
    - name: fibcalc
      rules:
        - alert: FibcalcDown
          expr: up{job="fibcalc"} == 0
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "Fibcalc is down"
            
        - alert: FibcalcHighLatency
          expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{job="fibcalc"}[5m])) > 5
          for: 10m
          labels:
            severity: warning
          annotations:
            summary: "High latency on Fibcalc"
            
        - alert: FibcalcHighErrorRate
          expr: rate(http_requests_total{job="fibcalc",status=~"5.."}[5m]) / rate(http_requests_total{job="fibcalc"}[5m]) > 0.05
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High error rate on Fibcalc"
```

## Dépannage

### Pod ne démarre pas

```bash
# Décrire le pod
kubectl describe pod -l app=fibcalc -n fibcalc

# Événements
kubectl get events -n fibcalc --sort-by='.lastTimestamp'

# Logs du pod précédent (si crash)
kubectl logs -l app=fibcalc -n fibcalc --previous
```

### Problèmes de ressources

```bash
# Métriques de ressources
kubectl top pods -n fibcalc

# Ajuster les limites
kubectl set resources deployment/fibcalc -n fibcalc \
  --limits=cpu=4000m,memory=4Gi \
  --requests=cpu=1000m,memory=1Gi
```

### Problèmes réseau

```bash
# Tester la connectivité
kubectl run test --rm -it --image=busybox -n fibcalc -- wget -qO- http://fibcalc/health

# Vérifier les endpoints
kubectl get endpoints fibcalc -n fibcalc
```

## Meilleures Pratiques

1. **Haute Disponibilité** : Utilisez au moins 2 réplicas avec PodDisruptionBudget
2. **Autoscaling** : Configurez HPA basé sur CPU/mémoire
3. **Sécurité** : Utilisez NetworkPolicy et SecurityContext
4. **Observabilité** : Activez les métriques Prometheus
5. **Ressources** : Définissez toujours requests et limits
6. **Mise à jour** : Utilisez RollingUpdate avec maxUnavailable: 0

