test-postman:
    # minikube tunnel
    # write to /etc/hosts "127.0.0.1 arch.homework"
	newman run tests/postman/hw4_tests.postman_collection.json
	newman run tests/postman/hw5_tests.postman_collection.json

install-postgres:
	./build/k8s/infra/postgresql/install.sh
	echo "sleep 30 sec to wait postgresql"
	sleep 30

build-app:
	docker login
	GOOS=linux GOARCH=amd64 go build -o bin/service cmd/main.go
	docker build --platform linux/amd64 --tag soundsofanarchy/otus-microservices-architect:v1.0.0 -f build/Dockerfile .
	docker push soundsofanarchy/otus-microservices-architect:v1.0.0

	GOOS=linux GOARCH=arm64 go build -o bin/service cmd/main.go
	docker build --platform linux/arm64 --tag soundsofanarchy/otus-microservices-architect:v1.0.0-arm -f build/Dockerfile .
	docker push soundsofanarchy/otus-microservices-architect:v1.0.0-arm

build-migrations:
	docker login
	docker build --platform linux/amd64 --tag soundsofanarchy/otus-microservices-architect:v1.0.0-migrations -f build/k8s/migrations/Dockerfile .
	docker build --platform linux/arm64 --tag soundsofanarchy/otus-microservices-architect:v1.0.0-migrations-arm -f build/k8s/migrations/Dockerfile .
	docker push soundsofanarchy/otus-microservices-architect:v1.0.0-migrations
	docker push soundsofanarchy/otus-microservices-architect:v1.0.0-migrations-arm

build-all: build-app build-migrations

run-migrations:
	kubectl apply -f ./build/k8s/migrations/secret.yaml
	kubectl apply -f ./build/k8s/migrations/job.yaml

delete-migrations:
	kubectl delete -f ./build/k8s/migrations/job.yaml

install-infra:
	./build/k8s/infra/install.sh
	echo "wait 30 sec for ingress"
	sleep 30

install-app:
	./build/k8s/install.sh

delete-app:
	./build/k8s/delete.sh

delete-all:
	./build/k8s/delete.sh
	./build/k8s/infra/delete.sh

encrypt-secret:
	gpg --batch --output build/k8s/migrations/secret.yaml.gpg --passphrase mypassword --symmetric build/k8s/migrations/secret.yaml

decrypt-secret:
	gpg --batch --output build/k8s/migrations/secret.yaml --passphrase mypassword --decrypt build/k8s/migrations/secret.yaml.gpg

install-world: install-infra install-postgres run-migrations install-app

check-hw: decrypt-secret install-world test-postman
