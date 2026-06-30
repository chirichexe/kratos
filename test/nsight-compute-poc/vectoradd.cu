#include <cuda_runtime.h>

#include <cmath>
#include <cstdlib>
#include <iostream>
#include <memory>

namespace {

constexpr int kElementCount = 1 << 20;
constexpr int kThreadsPerBlock = 256;

void checkCuda(cudaError_t result, const char* operation) {
  if (result != cudaSuccess) {
    std::cerr << operation << " failed: " << cudaGetErrorString(result) << '\n';
    std::exit(1);
  }
}

__global__ void vectorAdd(const float* a, const float* b, float* out, int n) {
  int idx = blockIdx.x * blockDim.x + threadIdx.x;
  if (idx < n) {
    out[idx] = a[idx] + b[idx];
  }
}

}  // namespace

int main() {
  std::cout << "Launching vectorAdd kernel with " << kElementCount
            << " elements\n";

  auto hostA = std::make_unique<float[]>(kElementCount);
  auto hostB = std::make_unique<float[]>(kElementCount);
  auto hostOut = std::make_unique<float[]>(kElementCount);

  for (int i = 0; i < kElementCount; ++i) {
    hostA[i] = static_cast<float>(i);
    hostB[i] = static_cast<float>(kElementCount - i);
  }

  float* deviceA = nullptr;
  float* deviceB = nullptr;
  float* deviceOut = nullptr;
  const size_t bytes = kElementCount * sizeof(float);

  checkCuda(cudaMalloc(&deviceA, bytes), "cudaMalloc deviceA");
  checkCuda(cudaMalloc(&deviceB, bytes), "cudaMalloc deviceB");
  checkCuda(cudaMalloc(&deviceOut, bytes), "cudaMalloc deviceOut");

  checkCuda(cudaMemcpy(deviceA, hostA.get(), bytes, cudaMemcpyHostToDevice),
            "cudaMemcpy hostA to deviceA");
  checkCuda(cudaMemcpy(deviceB, hostB.get(), bytes, cudaMemcpyHostToDevice),
            "cudaMemcpy hostB to deviceB");

  const int blocks = (kElementCount + kThreadsPerBlock - 1) / kThreadsPerBlock;
  vectorAdd<<<blocks, kThreadsPerBlock>>>(deviceA, deviceB, deviceOut,
                                          kElementCount);
  checkCuda(cudaGetLastError(), "vectorAdd launch");
  checkCuda(cudaDeviceSynchronize(), "cudaDeviceSynchronize");

  checkCuda(cudaMemcpy(hostOut.get(), deviceOut, bytes, cudaMemcpyDeviceToHost),
            "cudaMemcpy deviceOut to hostOut");

  for (int i = 0; i < kElementCount; ++i) {
    const float expected = hostA[i] + hostB[i];
    if (std::fabs(hostOut[i] - expected) > 0.001f) {
      std::cerr << "Validation failed at index " << i << ": expected "
                << expected << ", got " << hostOut[i] << '\n';
      return 1;
    }
  }

  checkCuda(cudaFree(deviceA), "cudaFree deviceA");
  checkCuda(cudaFree(deviceB), "cudaFree deviceB");
  checkCuda(cudaFree(deviceOut), "cudaFree deviceOut");

  std::cout << "Validation passed\n";
  return 0;
}
