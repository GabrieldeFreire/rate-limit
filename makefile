COMPOSE_RUN = docker compose up --build -V
COMPOSE_DOWN = docker compose down --volumes

run:
	bash -c "trap '$(COMPOSE_DOWN)' EXIT; $(COMPOSE_RUN)"

test-integration:
	go test -cover -coverpkg=./... -tags=integration -v
