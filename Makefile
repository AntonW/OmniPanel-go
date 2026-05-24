CONTAINER_DATE = $(shell date --rfc-3339=date)
CONTAINER_GIT = 000000
CONTAINER_VERSION ?= latest
CONTAINER_REGISTRY ?= docker.repo.org/atwi/
CONTAINER_REGISTRY_HOST = $(shell echo $(CONTAINER_REGISTRY) | cut -d/ -f1)

.PHONY: build-dev build clean login

login:
	@if [ -n "$(REGISTRY_USERNAME)" ] && [ -n "$(REGISTRY_TOKEN)" ]; then \
		echo "Logging in to $(CONTAINER_REGISTRY_HOST)..." && \
		mkdir -p ~/.docker && \
		echo "{\"auths\":{\"$(CONTAINER_REGISTRY_HOST)\":{\"auth\":\"$$(echo -n '$(REGISTRY_USERNAME):$(REGISTRY_TOKEN)' | base64)\"}}}" > ~/.docker/config.json; \
	else \
		echo "Skipping registry login (REGISTRY_USERNAME/REGISTRY_TOKEN not set)"; \
	fi

build: login
	@echo "Build OmniPanel-go serve"
	rm -rf kodata/ starter/ && \
	mkdir -p starter kodata && \
	cp -rav user/ starter/ && \
	cp -rav static/ kodata/static && \
	cp config.json kodata/config.json && \
	cp -rav starter/ kodata/starter && \
	$(eval CONTAINER_GIT := $(shell git rev-parse HEAD)) \
	KO_DOCKER_REPO=$(CONTAINER_REGISTRY) \
	ko build \
	--image-label org.opencontainers.image.created=$(CONTAINER_DATE) \
	--image-label org.opencontainers.image.version=$(CONTAINER_VERSION) \
	--image-label org.opencontainers.image.revision=$(CONTAINER_GIT) \
	--image-label org.opencontainers.image.title=OmniPanel-go \
	--image-label org.opencontainers.image.vendor="Antonio Winkler" \
	--image-label org.opencontainers.image.authors="Antonio Winkler" \
	-B -t $(CONTAINER_VERSION),latest
