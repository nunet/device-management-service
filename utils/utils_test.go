package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestIsExecutorStrictlyContained(t *testing.T) {
	docker := types.Executor{types.ExecutorTypeDocker}
	firecracker := types.Executor{types.ExecutorTypeFirecracker}
	wasm := types.Executor{types.ExecutorTypeWasm}

	executors1 := []interface{}{docker, firecracker, wasm}
	executors2 := []interface{}{docker, firecracker}
	executors3 := []interface{}{docker}

	// possitive assertions
	assert.True(t, IsStrictlyContained(executors1, executors2), fmt.Sprintf("Executors %s strictly contains executors %s", executors1, executors2))
	assert.True(t, IsStrictlyContained(executors2, executors3), fmt.Sprintf("Executors %s strictly contains executors %s", executors2, executors3))

	// negative assertions
	assert.False(t, IsStrictlyContained(executors2, executors1), fmt.Sprintf("Executors %s strictly contains executors %s", executors2, executors1))
}

func TestIntersectionStringSlices(t *testing.T) {
	docker := types.Executor{types.ExecutorTypeDocker}
	firecracker := types.Executor{types.ExecutorTypeFirecracker}
	wasm := types.Executor{types.ExecutorTypeWasm}

	executors1 := []interface{}{docker, firecracker, wasm}
	executors2 := []interface{}{docker, firecracker}
	executors3 := []interface{}{docker}
	executors4 := []interface{}{wasm}
	executors5 := []interface{}{firecracker, wasm}
	executors6 := []interface{}{docker, wasm}
	executors7 := []interface{}{firecracker}

	// positive assertions
	assert.Equal(t, IntersectionSlices(executors1, executors2), executors2)
	assert.Equal(t, IntersectionSlices(executors5, executors6), executors4)
	assert.Equal(t, IntersectionSlices(executors2, executors7), executors7)

	// negative assertions
	assert.NotEqual(t, IntersectionSlices(executors1, executors2), executors3)
}

func TestIsSameShallowType(t *testing.T) {
	var v1 = make(map[string]int)
	v1["1"] = 1
	v1["2"] = 2
	v1["3"] = 3

	var v4 = make(map[string]int)
	v4["1"] = 5
	v4["2"] = 6
	v4["3"] = 7

	v2 := "string"
	v5 := "another string"
	v3 := float32(6.629)
	var v6 float32 = 7.9790

	var v7 = make(map[string]interface{})
	v7["1"] = 1
	v7["2"] = 3
	v7["3"] = 3

	var v8 = make(map[string]interface{})
	v8["1"] = 5
	v8["2"] = 6
	v8["3"] = 7

	assert.True(t, IsSameShallowType(v1, v4))
	assert.True(t, IsSameShallowType(v2, v5))
	assert.True(t, IsSameShallowType(v3, v6))
	assert.True(t, IsSameShallowType(v7, v8))

	assert.False(t, IsSameShallowType(v1, v2))
	assert.False(t, IsSameShallowType(v2, v3))
}

func TestIsExecutor(t *testing.T) {
	executor1 := types.Executor{types.ExecutorTypeDocker}
	executor2 := types.Executor{types.ExecutorTypeFirecracker}
	executor3 := types.Executor{types.ExecutorTypeWasm}

	// positive assertions
	assert.True(t, IsExecutor(executor1))
	assert.True(t, IsExecutor(executor2))
	assert.True(t, IsExecutor(executor3))

	// negative assertions
	assert.False(t, IsExecutor("string"))
	assert.False(t, IsExecutor(1))
}

func TestIsExecutorType(t *testing.T) {
	var executorType1 types.ExecutorType = types.ExecutorTypeDocker
	var executorType2 types.ExecutorType = types.ExecutorTypeFirecracker
	var executorType3 types.ExecutorType = types.ExecutorTypeWasm

	// positive assertions
	assert.True(t, IsExecutorType(executorType1))
	assert.True(t, IsExecutorType(executorType2))
	assert.True(t, IsExecutorType(executorType3))

	// negative assertions
	assert.False(t, IsExecutorType("string"))
	assert.False(t, IsExecutorType(1))
}

func TestIsJobTypes(t *testing.T) {
	var jobType1 types.JobTypes
	jobType1 = append(jobType1, types.BATCH)
	jobType1 = append(jobType1, types.SINGLERUN)

	var jobType2 types.JobTypes
	jobType2 = append(jobType2, types.BATCH)
	jobType2 = append(jobType2, types.LONGRUNNING)

	var jobType3 types.JobTypes
	jobType3 = append(jobType3, types.RECURRING)
	jobType3 = append(jobType3, types.SINGLERUN)

	// positive assertions
	assert.True(t, IsJobTypes(jobType1))
	assert.True(t, IsJobTypes(jobType2))
	assert.True(t, IsJobTypes(jobType3))

	// negative assertions
	assert.False(t, IsJobTypes("string"))
	assert.False(t, IsJobTypes(1))
}

func TestIsJobType(t *testing.T) {
	var jobType1 types.JobType = types.BATCH
	var jobType2 types.JobType = types.SINGLERUN
	var jobType3 types.JobType = types.LONGRUNNING

	// positive assertions
	assert.True(t, IsJobType(jobType1))
	assert.True(t, IsJobType(jobType2))
	assert.True(t, IsJobType(jobType3))

	// negative assertions
	assert.False(t, IsJobType("string"))
	assert.False(t, IsJobType(1))
}

func TestConvertTypedSliceToUntypedSlice(t *testing.T) {
	var jobType1 types.JobTypes
	jobType1 = append(jobType1, types.BATCH)
	jobType1 = append(jobType1, types.SINGLERUN)

	var jobType2 types.JobTypes
	jobType2 = append(jobType2, types.BATCH)
	jobType2 = append(jobType2, types.LONGRUNNING)

	var jobType3 types.JobTypes
	jobType3 = append(jobType3, types.RECURRING)
	jobType3 = append(jobType3, types.SINGLERUN)

	// positive assertions
	actualValue := ConvertTypedSliceToUntypedSlice(jobType1)
	expectedValue := []interface{}{types.BATCH, types.SINGLERUN}
	assert.Equal(t, expectedValue, actualValue)

	actualValue = ConvertTypedSliceToUntypedSlice(jobType2)
	expectedValue = []interface{}{types.BATCH, types.LONGRUNNING}
	assert.Equal(t, expectedValue, actualValue)
	
	
	actualValue = ConvertTypedSliceToUntypedSlice(jobType3)
	expectedValue = []interface{}{types.RECURRING, types.SINGLERUN}	
	assert.Equal(t, expectedValue, actualValue)
}

func TestIsGPUVendor(t *testing.T) {
	var gpuVendor1 types.GPUVendor = types.GPUVendorNvidia
	var gpuVendor2 types.GPUVendor = types.GPUVendorAMDATI
	var gpuVendor3 types.GPUVendor = types.GPUVendorIntel

	// positive assertions
	assert.True(t, IsGPUVendor(gpuVendor1))
	assert.True(t, IsGPUVendor(gpuVendor2))
	assert.True(t, IsGPUVendor(gpuVendor3))

	// negative assertions
	assert.False(t, IsGPUVendor("string"))
	assert.False(t, IsGPUVendor(1))
}
