#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "deploy namespace"
kubectl apply -f build/k8s/order-service/namespace.yaml

echo "apply app components"
kubectl apply -f build/k8s/order-service
