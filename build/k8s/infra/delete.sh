#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "delete nginx"
helm -n m delete nginx
kubectl delete namespace m

echo "delete prom"
helm delete prometheus
echo "delete grafana"
helm delete grafana
echo "delete kafka"
helm -n kafka delete kafka
kubectl delete namespace kafka
echo "delete keycloak"
helm -n kafka delete keycloak
kubectl delete namespace keycloak