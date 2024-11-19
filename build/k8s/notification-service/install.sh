#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "deploy namespace"
kubectl apply -f build/k8s/notification-service/namespace.yaml

echo "apply app components"
kubectl apply -f build/k8s/notification-service