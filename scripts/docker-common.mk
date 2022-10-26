##@ Common docker build targets
.PHONY: docker-command-check ## check if the docker command is defined in the path
docker-command-check:
	 @command -v ${DOCKER_CMD} > /dev/null || \
		(echo "ERROR: no ${DOCKER_CMD} in path, it must be installed " && exit 1)

.PHONY: docker-build
docker-build: docker-command-check ## Build docker image containing the executable.
	${DOCKER_CMD} build -t ${IMAGE_TAG_BASE}:v$(VERSION) .

.PHONY: docker-build-pod
docker-build-pod: docker-command-check ## Build docker image containing the executable.
	cd ../ && ${DOCKER_CMD} build -f ${DOCKERFILE} -t ${IMAGE_TAG_BASE}:v$(VERSION) .

.PHONY: docker-push
docker-push: docker-command-check ## Push docker image containing the executable.
	${DOCKER_CMD} push ${IMAGE_TAG_BASE}:v$(VERSION)
