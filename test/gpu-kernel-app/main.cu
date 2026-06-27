#include <cuda_runtime.h>

#include <cmath>
#include <cstdlib>
#include <iostream>
#include <memory>
#include <string>

namespace {

constexpr int kThreadsPerBlock = 256;

void checkCuda(cudaError_t result, const char* operation) {
  if (result != cudaSuccess) {
    std::cerr << operation << " failed: " << cudaGetErrorString(result) << '\n';
    std::exit(1);
  }
}

struct CudaDeleter {
  void operator()(float* ptr) const {
    if (ptr != nullptr) {
      cudaFree(ptr);
    }
  }
};

using DeviceBuffer = std::unique_ptr<float, CudaDeleter>;

DeviceBuffer allocateDeviceBuffer(size_t elements) {
  float* ptr = nullptr;
  checkCuda(cudaMalloc(&ptr, elements * sizeof(float)), "cudaMalloc");
  return DeviceBuffer(ptr);
}

int envInt(const char* name, int fallback) {
  const char* raw = std::getenv(name);
  if (raw == nullptr || raw[0] == '\0') {
    return fallback;
  }
  return std::atoi(raw);
}

template <typename Launcher>
float timeKernel(Launcher launcher, cudaStream_t stream) {
  cudaEvent_t start;
  cudaEvent_t stop;
  checkCuda(cudaEventCreate(&start), "cudaEventCreate start");
  checkCuda(cudaEventCreate(&stop), "cudaEventCreate stop");

  checkCuda(cudaEventRecord(start, stream), "cudaEventRecord start");
  launcher(stream);
  checkCuda(cudaEventRecord(stop, stream), "cudaEventRecord stop");
  checkCuda(cudaEventSynchronize(stop), "cudaEventSynchronize stop");

  float milliseconds = 0.0f;
  checkCuda(cudaEventElapsedTime(&milliseconds, start, stop),
            "cudaEventElapsedTime");
  checkCuda(cudaEventDestroy(start), "cudaEventDestroy start");
  checkCuda(cudaEventDestroy(stop), "cudaEventDestroy stop");
  return milliseconds;
}

__global__ void vectorAdd(const float* a, const float* b, float* out, int n) {
  int idx = blockIdx.x * blockDim.x + threadIdx.x;
  if (idx < n) {
    out[idx] = a[idx] + b[idx];
  }
}

__global__ void computeBound(float* out, int n, int iterations) {
  int idx = blockIdx.x * blockDim.x + threadIdx.x;
  if (idx >= n) {
    return;
  }

  float value = static_cast<float>((idx % 251) + 1) * 0.001f;
  float acc = value;
  for (int i = 0; i < iterations; ++i) {
    acc = fmaf(acc, 1.000001f, value);
    acc = fmaf(acc, 0.999999f, -value);
    acc = fmaf(acc, 1.000003f, 0.000001f);
    acc = fmaf(acc, 0.999997f, -0.000001f);
  }
  out[idx] = acc;
}

__global__ void memoryBound(const float* a, const float* b, float* out, int n,
                            int iterations) {
  int idx = blockIdx.x * blockDim.x + threadIdx.x;
  if (idx >= n) {
    return;
  }

  float value = out[idx];
  for (int i = 0; i < iterations; ++i) {
    value = a[idx] + 1.6180339f * b[idx] + value * 0.000001f;
  }
  out[idx] = value;
}

int runVectorAdd() {
  constexpr int kElementCount = 4096;
  std::cout << "Launching vectorAdd kernel with " << kElementCount
            << " elements\n";

  auto hostA = std::make_unique<float[]>(kElementCount);
  auto hostB = std::make_unique<float[]>(kElementCount);
  auto hostOut = std::make_unique<float[]>(kElementCount);

  for (int i = 0; i < kElementCount; ++i) {
    hostA[i] = static_cast<float>(i);
    hostB[i] = static_cast<float>(kElementCount - i);
  }

  DeviceBuffer deviceA = allocateDeviceBuffer(kElementCount);
  DeviceBuffer deviceB = allocateDeviceBuffer(kElementCount);
  DeviceBuffer deviceOut = allocateDeviceBuffer(kElementCount);

  const size_t bytes = kElementCount * sizeof(float);
  checkCuda(cudaMemcpy(deviceA.get(), hostA.get(), bytes, cudaMemcpyHostToDevice),
            "cudaMemcpy hostA to deviceA");
  checkCuda(cudaMemcpy(deviceB.get(), hostB.get(), bytes, cudaMemcpyHostToDevice),
            "cudaMemcpy hostB to deviceB");

  const int blocks = (kElementCount + kThreadsPerBlock - 1) / kThreadsPerBlock;
  vectorAdd<<<blocks, kThreadsPerBlock>>>(deviceA.get(), deviceB.get(),
                                          deviceOut.get(), kElementCount);
  checkCuda(cudaGetLastError(), "vectorAdd launch");
  checkCuda(cudaDeviceSynchronize(), "cudaDeviceSynchronize");

  checkCuda(cudaMemcpy(hostOut.get(), deviceOut.get(), bytes, cudaMemcpyDeviceToHost),
            "cudaMemcpy deviceOut to hostOut");

  for (int i = 0; i < kElementCount; ++i) {
    const float expected = hostA[i] + hostB[i];
    if (std::fabs(hostOut[i] - expected) > 0.001f) {
      std::cerr << "Validation failed at index " << i << ": expected "
                << expected << ", got " << hostOut[i] << '\n';
      return 1;
    }
  }

  std::cout << "GPU kernel completed successfully\n";
  std::cout << "Validation passed\n";
  return 0;
}

int runComputeBound() {
  const int elements = envInt("KRATOS_ELEMENTS", 1 << 20);
  const int iterations = envInt("KRATOS_ITERATIONS", 4096);
  const int blocks = (elements + kThreadsPerBlock - 1) / kThreadsPerBlock;

  DeviceBuffer out = allocateDeviceBuffer(elements);
  checkCuda(cudaMemset(out.get(), 0, elements * sizeof(float)), "cudaMemset out");

  cudaStream_t stream;
  checkCuda(cudaStreamCreate(&stream), "cudaStreamCreate");
  const float elapsedMs = timeKernel(
      [&](cudaStream_t timedStream) {
        computeBound<<<blocks, kThreadsPerBlock, 0, timedStream>>>(
            out.get(), elements, iterations);
        checkCuda(cudaGetLastError(), "computeBound launch");
      },
      stream);
  checkCuda(cudaStreamDestroy(stream), "cudaStreamDestroy");

  auto sample = std::make_unique<float[]>(1);
  checkCuda(cudaMemcpy(sample.get(), out.get(), sizeof(float),
                       cudaMemcpyDeviceToHost),
            "cudaMemcpy compute sample");

  const double fmasPerElement = static_cast<double>(iterations) * 4.0;
  const double flops = static_cast<double>(elements) * fmasPerElement * 2.0;
  const double gflops = flops / (static_cast<double>(elapsedMs) / 1000.0) / 1e9;

  std::cout << "workload=compute-bound\n";
  std::cout << "elements=" << elements << '\n';
  std::cout << "iterations=" << iterations << '\n';
  std::cout << "elapsed_ms=" << elapsedMs << '\n';
  std::cout << "estimated_gflops=" << gflops << '\n';
  std::cout << "sample=" << sample[0] << '\n';
  return 0;
}

int runMemoryBound() {
  const int elements = envInt("KRATOS_ELEMENTS", 8 << 20);
  const int iterations = envInt("KRATOS_ITERATIONS", 128);
  const int blocks = (elements + kThreadsPerBlock - 1) / kThreadsPerBlock;

  DeviceBuffer a = allocateDeviceBuffer(elements);
  DeviceBuffer b = allocateDeviceBuffer(elements);
  DeviceBuffer out = allocateDeviceBuffer(elements);
  checkCuda(cudaMemset(a.get(), 1, elements * sizeof(float)), "cudaMemset a");
  checkCuda(cudaMemset(b.get(), 2, elements * sizeof(float)), "cudaMemset b");
  checkCuda(cudaMemset(out.get(), 0, elements * sizeof(float)), "cudaMemset out");

  cudaStream_t stream;
  checkCuda(cudaStreamCreate(&stream), "cudaStreamCreate");
  const float elapsedMs = timeKernel(
      [&](cudaStream_t timedStream) {
        memoryBound<<<blocks, kThreadsPerBlock, 0, timedStream>>>(
            a.get(), b.get(), out.get(), elements, iterations);
        checkCuda(cudaGetLastError(), "memoryBound launch");
      },
      stream);
  checkCuda(cudaStreamDestroy(stream), "cudaStreamDestroy");

  auto sample = std::make_unique<float[]>(1);
  checkCuda(cudaMemcpy(sample.get(), out.get(), sizeof(float),
                       cudaMemcpyDeviceToHost),
            "cudaMemcpy memory sample");

  const double bytesPerIteration = static_cast<double>(elements) * sizeof(float) * 3.0;
  const double totalBytes = bytesPerIteration * static_cast<double>(iterations);
  const double gibPerSecond =
      totalBytes / (static_cast<double>(elapsedMs) / 1000.0) / (1024.0 * 1024.0 * 1024.0);

  std::cout << "workload=memory-bound\n";
  std::cout << "elements=" << elements << '\n';
  std::cout << "iterations=" << iterations << '\n';
  std::cout << "elapsed_ms=" << elapsedMs << '\n';
  std::cout << "estimated_gib_per_second=" << gibPerSecond << '\n';
  std::cout << "sample=" << sample[0] << '\n';
  return 0;
}

void printUsage(const char* argv0) {
  std::cerr << "Usage: " << argv0 << " [vectoradd|compute|memory]\n";
  std::cerr << "Optional env: KRATOS_ELEMENTS, KRATOS_ITERATIONS\n";
}

}  // namespace

int main(int argc, char** argv) {
  const std::string mode = argc > 1 ? argv[1] : "vectoradd";
  if (mode == "vectoradd") {
    return runVectorAdd();
  }
  if (mode == "compute") {
    return runComputeBound();
  }
  if (mode == "memory") {
    return runMemoryBound();
  }
  printUsage(argv[0]);
  return 2;
}
