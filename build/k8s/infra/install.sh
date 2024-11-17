#!/bin/bash

echo "switching to minikube cluster"
kubectx minikube

echo "deploy nginx ingress controller"
kubectl create namespace m && helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx/ && helm repo update && helm install nginx ingress-nginx/ingress-nginx --namespace m -f build/k8s/infra/nginx-ingress-controller/values_nginx_ingress.yaml

echo "install prometheus"
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install prometheus prometheus-community/prometheus
helm upgrade -i prometheus prometheus-community/prometheus -f build/k8s/infra/prometheus/values.yaml

echo "install grafana"
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm upgrade -i grafana grafana/grafana -f build/k8s/infra/grafana/values.yaml

echo "install kafka"
kubectl create namespace kafka || true
helm upgrade -n kafka -i kafka oci://registry-1.docker.io/bitnamicharts/kafka -f build/k8s/infra/kafka/values.yaml
