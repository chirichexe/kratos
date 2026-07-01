IMAGE ?= kratos-nsight-compute-poc:latest
CLUSTER ?= kratos-gpu
GPUS ?= all

.PHONY: build run load

build:
	docker build -f Dockerfile.runtime -t $(IMAGE) ../..

run:
	docker run --rm --gpus $(GPUS) --cap-add SYS_ADMIN $(IMAGE)

load:
	kind load docker-image $(IMAGE) --name $(CLUSTER)
