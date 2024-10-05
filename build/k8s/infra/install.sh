#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "deploy nginx ingress controller"
kubectl create namespace m && helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx/ && helm repo update && helm install nginx ingress-nginx/ingress-nginx --namespace m -f build/k8s/infra/nginx-ingress-controller/values_nginx_ingress.yaml