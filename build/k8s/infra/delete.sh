#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "delete"
kubectl delete namespace m