IMAGE ?= kratos-gpu-kernel-test:latest
GPUS ?= all

.PHONY: build run run-compute run-memory

build:
	docker build -t $(IMAGE) .

run:
	docker run --rm --gpus $(GPUS) $(IMAGE)

run-compute:
	docker run --rm --gpus $(GPUS) $(IMAGE) compute

run-memory:
	docker run --rm --gpus $(GPUS) $(IMAGE) memory
