IMAGE ?= kratos-nsight-compute-poc:latest
CLUSTER ?= kratos-gpu
JOB ?= nsight-compute-vectoradd
SIDECAR_JOB ?= nsight-compute-vectoradd-sidecar
NVIDIA_SAMPLE_SIDECAR_JOB ?= nsight-compute-nvidia-sample-sidecar
GPUS ?= all

.PHONY: build run load apply apply-sidecar apply-nvidia-sample-sidecar logs logs-sidecar logs-nvidia-sample-sidecar clean clean-sidecar clean-nvidia-sample-sidecar

build:
	docker build -t $(IMAGE) .

run:
	docker run --rm --gpus $(GPUS) --cap-add SYS_ADMIN $(IMAGE)

load:
	kind load docker-image $(IMAGE) --name $(CLUSTER)

apply:
	kubectl apply -f job.yaml

apply-sidecar:
	kubectl apply -f job-sidecar.yaml

apply-nvidia-sample-sidecar:
	kubectl apply -f job-sidecar-nvidia-sample.yaml

logs:
	kubectl logs job/$(JOB)

logs-sidecar:
	kubectl logs job/$(SIDECAR_JOB) -c nsight-compute-sidecar

logs-nvidia-sample-sidecar:
	kubectl logs job/$(NVIDIA_SAMPLE_SIDECAR_JOB) -c ncu-sidecar

clean:
	kubectl delete job $(JOB) --ignore-not-found

clean-sidecar:
	kubectl delete job $(SIDECAR_JOB) --ignore-not-found

clean-nvidia-sample-sidecar:
	kubectl delete job $(NVIDIA_SAMPLE_SIDECAR_JOB) --ignore-not-found
