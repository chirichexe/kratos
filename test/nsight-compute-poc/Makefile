IMAGE ?= kratos-nsight-compute-poc:latest
CLUSTER ?= kratos-gpu
JOB ?= nsight-compute-vectoradd
RUNNER_JOB ?= nsight-compute-vectoradd-runner
NVIDIA_SAMPLE_RUNNER_JOB ?= nsight-compute-nvidia-sample-runner
GPUS ?= all

.PHONY: build run load apply apply-runner apply-nvidia-sample-runner logs logs-runner logs-nvidia-sample-runner clean clean-runner clean-nvidia-sample-runner

build:
	docker build -t $(IMAGE) .

run:
	docker run --rm --gpus $(GPUS) --cap-add SYS_ADMIN $(IMAGE)

load:
	kind load docker-image $(IMAGE) --name $(CLUSTER)

apply:
	kubectl apply -f job.yaml

apply-runner:
	kubectl apply -f job-profiling-runner.yaml

apply-nvidia-sample-runner:
	kubectl apply -f job-profiling-runner-nvidia-sample.yaml

logs:
	kubectl logs job/$(JOB)

logs-runner:
	kubectl logs job/$(RUNNER_JOB) -c profiling-runner

logs-nvidia-sample-runner:
	kubectl logs job/$(NVIDIA_SAMPLE_RUNNER_JOB) -c profiling-runner

clean:
	kubectl delete job $(JOB) --ignore-not-found

clean-runner:
	kubectl delete job $(RUNNER_JOB) --ignore-not-found

clean-nvidia-sample-runner:
	kubectl delete job $(NVIDIA_SAMPLE_RUNNER_JOB) --ignore-not-found
