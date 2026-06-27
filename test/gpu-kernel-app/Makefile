IMAGE ?= kratos-gpu-kernel-test:latest
GPUS ?= all

.PHONY: build run

build:
	docker build -t $(IMAGE) .

run:
	docker run --rm --gpus $(GPUS) $(IMAGE)
