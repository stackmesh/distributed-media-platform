.PHONY: test test-gateway test-billing up

test: test-gateway test-billing

test-gateway:
	cd go-media-gateway && go test ./...

test-billing:
	cd java-billing-engine && mvn -B test

up:
	docker compose up --build
