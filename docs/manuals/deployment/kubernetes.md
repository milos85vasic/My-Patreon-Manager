# Deployment: Kubernetes

## 1. Build and push the image

```bash
podman build -t ghcr.io/milos85vasic/patreon-manager:latest .
podman push ghcr.io/milos85vasic/patreon-manager:latest
```

## 2. Create the namespace and secret

```bash
kubectl create namespace patreon-manager
kubectl -n patreon-manager create secret generic pm-env --from-env-file=.env
```

## 3. Deployment manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: patreon-manager
  namespace: patreon-manager
spec:
  replicas: 1
  selector:
    matchLabels:
      app: patreon-manager
  template:
    metadata:
      labels:
        app: patreon-manager
    spec:
      containers:
        - name: server
          image: ghcr.io/milos85vasic/patreon-manager:latest
          ports:
            - containerPort: 8080
          envFrom:
            - secretRef:
                name: pm-env
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
          resources:
            requests:
              memory: "128Mi"
              cpu: "100m"
            limits:
              memory: "512Mi"
              cpu: "500m"
---
apiVersion: v1
kind: Service
metadata:
  name: patreon-manager
  namespace: patreon-manager
spec:
  selector:
    app: patreon-manager
  ports:
    - port: 8080
      targetPort: 8080
```

## 4. Apply

```bash
kubectl apply -f k8s/deployment.yaml
kubectl -n patreon-manager get pods
```

## 5. Run CLI as a Job

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: pm-sync
  namespace: patreon-manager
spec:
  template:
    spec:
      containers:
        - name: sync
          image: ghcr.io/milos85vasic/patreon-manager:latest
          command: ["patreon-manager", "sync", "--dry-run"]
          envFrom:
            - secretRef:
                name: pm-env
      restartPolicy: Never
```
