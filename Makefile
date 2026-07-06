.PHONY: build test fmt web-build clean minikube-deploy minikube-smoke

build:
	mkdir -p bin
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./...
	cd web && npm ci && npm test

fmt:
	gofmt -w cmd internal

web-build:
	cd web && npm ci && npm run build

clean:
	rm -rf bin web/dist

minikube-deploy:
	./scripts/minikube-deploy.sh

minikube-smoke:
	./scripts/minikube-smoke.sh
