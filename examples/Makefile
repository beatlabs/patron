DOCKER_COMPOSE = docker-compose

default: e2e

# e2e triggers the examples-run and than stop the processes
e2e: examples-run
	$(MAKE) stop && echo "Successful E2E"

# runs the query that will flow across all of the services
examples-run: start-svcs
	sleep 5 && go run check/main.go

# starts all the example services
start-svcs: start-infra stop-svcs
	./services.sh start

# stops all the example services
stop-svcs:
	./services.sh stop && ./services.sh clean

# starts infrastructure (kafka, sqs, rabbitmq, etc) and wait for 10 seconds to make sure everything is ready
start-infra: stop-infra
	$(DOCKER_COMPOSE) up -d && sleep 10s

# stops infrastructure (kafka, sqs, rabbitmq, etc)
stop-infra:
	$(DOCKER_COMPOSE) down

# stops the infrastructure and the example services
stop: stop-svcs stop-infra

# starts the infrastructure and the example services
start: start-infra start-svcs

# disallow any parallelism (-j) for Make. This is necessary since some
# commands during the build process create temporary files that collide
# under parallel conditions.
.NOTPARALLEL:

.PHONY: e2e examples-run start-svcs start-svcs stop stop-svcs stop-infra start