/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import gpuv1alpha1 "github.com/chirichexe/kratos/api/v1alpha1"

const (
	executionJobNameSuffix            = "-execution"
	profileSummaryConfigMapNameSuffix = "-profile-summary"
	profilingJobNameSuffix            = "-profiling"
	workloadProfileNameSuffix         = "-profile"
)

func executionJobName(experiment *gpuv1alpha1.CUDAExperiment) string {
	return experiment.Name + executionJobNameSuffix
}

func profilingJobName(experiment *gpuv1alpha1.CUDAExperiment) string {
	return experiment.Name + profilingJobNameSuffix
}

func profileSummaryConfigMapName(experiment *gpuv1alpha1.CUDAExperiment) string {
	return experiment.Name + profileSummaryConfigMapNameSuffix
}

func workloadProfileName(experiment *gpuv1alpha1.CUDAExperiment) string {
	return experiment.Name + workloadProfileNameSuffix
}
