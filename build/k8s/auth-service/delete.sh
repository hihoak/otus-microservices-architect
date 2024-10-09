#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "delete namespace"
kubectl delete -f build/k8s/auth-service/namespace.yaml
