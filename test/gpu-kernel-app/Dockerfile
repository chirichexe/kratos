ARG CUDA_VERSION=12.5.1
ARG UBUNTU_VERSION=22.04

FROM nvidia/cuda:${CUDA_VERSION}-devel-ubuntu${UBUNTU_VERSION} AS build

WORKDIR /src
COPY main.cu .

RUN nvcc -O2 -std=c++17 -o /usr/local/bin/gpu-kernel main.cu

FROM nvidia/cuda:${CUDA_VERSION}-runtime-ubuntu${UBUNTU_VERSION}

COPY --from=build /usr/local/bin/gpu-kernel /usr/local/bin/gpu-kernel

ENTRYPOINT ["/usr/local/bin/gpu-kernel"]
