#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "create a namespace"
kubectl apply -f build/k8s/infra/postgresql/namespace.yaml

echo "create pv and pvc"
kubectl apply -f build/k8s/infra/postgresql/postgres-pv.yaml
kubectl apply -f build/k8s/infra/postgresql/postgres-pvc.yaml

echo "deploy postgresql"
helm upgrade -i -n app psql-test bitnami/postgresql --set persistence.existingClaim=postgresql-pv-claim --set volumePermissions.enabled=true --set global.postgresql.auth.postgresPassword="empty"
