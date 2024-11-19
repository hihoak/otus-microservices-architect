#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "delete namespace"
kubectl delete namespace --force app
