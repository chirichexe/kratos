ARG CUDA_IMAGE=nvcr.io/nvidia/cuda:13.3.0-devel-ubuntu24.04

FROM ${CUDA_IMAGE}

ARG NSIGHT_COMPUTE_PACKAGE=nsight-compute-2025.2.1

WORKDIR /src
COPY vectoradd.cu .
COPY run-profile.sh /usr/local/bin/run-profile

RUN apt-get update \
    && if ! command -v ncu >/dev/null 2>&1; then \
      apt-get install -y --no-install-recommends "${NSIGHT_COMPUTE_PACKAGE}"; \
    fi \
    && rm -rf /var/lib/apt/lists/* \
    && nvcc -O2 -lineinfo -std=c++17 -o /usr/local/bin/vectoradd vectoradd.cu \
    && chmod +x /usr/local/bin/run-profile \
    && if ! command -v ncu >/dev/null 2>&1; then \
      ncu_path="$(find /opt/nvidia -path '*nsight-compute*/ncu' -type f | grep -v '/target/' | head -n 1)" \
      && printf '#!/bin/sh\nexec "%s" "$@"\n' "${ncu_path}" > /usr/local/bin/ncu \
      && chmod +x /usr/local/bin/ncu; \
    fi \
    && command -v ncu

ENTRYPOINT ["/usr/local/bin/run-profile"]
