IMAGE ?= kratos-nsight-compute-poc:latest
CLUSTER ?= kratos-gpu
JOB ?= nsight-compute-vectoradd
GPUS ?= all

.PHONY: build run load apply logs clean

build:
	docker build -t $(IMAGE) .

run:
	docker run --rm --gpus $(GPUS) --cap-add SYS_ADMIN $(IMAGE)

load:
	kind load docker-image $(IMAGE) --name $(CLUSTER)

apply:
	kubectl apply -f job.yaml

logs:
	kubectl logs job/$(JOB)

clean:
	kubectl delete job $(JOB) --ignore-not-found
