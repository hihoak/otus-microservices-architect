test-postman:
    # minikube tunnel
    # write to /etc/hosts "127.0.0.1 arch.homework"
	newman run tests/postman/*

install-all:
	./build/k8s/infra/install.sh
	./build/k8s/install.sh

install-app:
	./build/k8s/install.sh

delete-all:
	./build/k8s/delete.sh
	./build/k8s/infra/delete.sh