// Package hami provides integration with HAMi (Heterogeneous AI Computing
// Virtualization Middleware), a CNCF Sandbox project for GPU virtualization
// on Kubernetes. It supports NVIDIA, Ascend, MLU, DCU, and Hygon GPUs.
//
// Key features used:
//   - GPU memory hard isolation (MiB granularity)
//   - GPU compute core quota (percentage)
//   - Scheduling policies: Binpack / Spread / Topology-aware
//
// Reference: https://github.com/Project-HAMi/HAMi
package hami

// SchedulerPolicy controls how HAMi places vGPU workloads on physical GPUs.
type SchedulerPolicy string

const (
	// PolicyBinpack packs multiple workloads onto the fewest physical GPUs,
	// maximizing overall GPU utilization. Best for shared inference clusters.
	PolicyBinpack SchedulerPolicy = "binpack"

	// PolicySpread distributes workloads across different physical GPUs,
	// maximizing fault isolation. Best for dedicated endpoints.
	PolicySpread SchedulerPolicy = "spread"

	// PolicyTopologyAware selects GPUs with high inter-GPU bandwidth
	// (NVLink/NVSwitch). Best for MoE models with expert parallelism.
	PolicyTopologyAware SchedulerPolicy = "topology-aware"
)

// VendorType identifies the GPU hardware vendor.
type VendorType string

const (
	VendorNVIDIA    VendorType = "nvidia"
	VendorAscend    VendorType = "ascend"    // Huawei Ascend 910
	VendorCambricon VendorType = "cambricon" // MLU
	VendorHygon     VendorType = "hygon"     // DCU
)

// GPUResourceSpec defines the GPU resource requirements for a single container,
// following HAMi's three-dimensional resource declaration standard.
type GPUResourceSpec struct {
	// Count is the number of virtual GPU slices to request.
	// Maps to: nvidia.com/gpu (Kubernetes resource limit)
	Count int `json:"count"`

	// MemoryMiB is the GPU memory to reserve in MiB (hard isolation).
	// HAMi enforces this limit in hardware; exceeding it causes OOM.
	// Maps to: nvidia.com/gpumem
	// Common values: 8192 (8GB), 16384 (16GB), 20480 (20GB), 40960 (40GB)
	MemoryMiB int `json:"memory_mib"`

	// Cores is the GPU compute cores percentage (0–100).
	// 0 means no limit (full unrestricted access).
	// Maps to: nvidia.com/gpucores
	Cores int `json:"cores"`

	// Vendor specifies the GPU hardware vendor. Defaults to NVIDIA.
	Vendor VendorType `json:"vendor"`
}

// ScheduleRequest is sent to the HAMi scheduler extender to make
// a placement decision for a new vLLM worker pod.
type ScheduleRequest struct {
	// EndpointID is the dedicated endpoint ID requesting GPU placement.
	EndpointID string `json:"endpoint_id"`

	// ModelName is the LLM model to be served (e.g., "deepseek-ai/DeepSeek-V3").
	ModelName string `json:"model_name"`

	// GPUSpec defines the GPU resource requirements.
	GPUSpec GPUResourceSpec `json:"gpu_spec"`

	// Policy controls placement strategy.
	Policy SchedulerPolicy `json:"policy"`

	// Replicas is the number of worker pods to schedule.
	Replicas int `json:"replicas"`

	// TopologyAware enables NVLink topology optimization.
	// Only effective when Policy == PolicyTopologyAware.
	TopologyAware bool `json:"topology_aware"`
}

// ScheduleResult contains the placement decision from HAMi scheduler.
type ScheduleResult struct {
	// Scheduled indicates whether placement succeeded.
	Scheduled bool `json:"scheduled"`

	// NodeName is the Kubernetes node selected for placement.
	NodeName string `json:"node_name"`

	// PhysicalGPUID is the physical GPU index on the selected node.
	PhysicalGPUID int `json:"physical_gpu_id"`

	// AllocatedMemoryMiB is the actual memory allocated (may differ from request
	// if the scheduler applies admission control adjustments).
	AllocatedMemoryMiB int `json:"allocated_memory_mib"`

	// AllocatedCores is the actual core quota allocated.
	AllocatedCores int `json:"allocated_cores"`

	// Reason provides explanation when Scheduled == false.
	Reason string `json:"reason,omitempty"`
}

// NodeGPUInfo describes GPU resources available on a single Kubernetes node.
type NodeGPUInfo struct {
	NodeName      string     `json:"node_name"`
	Vendor        VendorType `json:"vendor"`
	GPUCount      int        `json:"gpu_count"`
	TotalMemMiB   int        `json:"total_mem_mib"`
	UsedMemMiB    int        `json:"used_mem_mib"`
	FreeMem       int        `json:"free_mem_mib"`
	Utilization   float64    `json:"utilization_pct"`
	HasNVLink     bool       `json:"has_nvlink"`
}

// HAMi resource name constants (Kubernetes extended resources)
const (
	ResourceGPU      = "nvidia.com/gpu"
	ResourceGPUMem   = "nvidia.com/gpumem"
	ResourceGPUCores = "nvidia.com/gpucores"

	// HAMi pod annotation for scheduler policy
	AnnotationSchedulerPolicy = "hami.io/gpu-scheduler-policy"
	// HAMi pod annotation for topology awareness
	AnnotationTopologyAware = "hami.io/gpu-topology-aware"
)

// DefaultGPUSpec returns sensible defaults for a vLLM worker GPU allocation.
// These can be overridden per-endpoint via the API.
func DefaultGPUSpec(modelSizeB float64) GPUResourceSpec {
	// Heuristic: ~2GB per 1B parameters at FP16, plus 20% overhead
	memMiB := int(modelSizeB * 2.0 * 1024 * 1.2)
	if memMiB < 8192 {
		memMiB = 8192 // minimum 8GB
	}
	if memMiB > 81920 {
		memMiB = 81920 // cap at 80GB (A100/H100 single GPU)
	}
	return GPUResourceSpec{
		Count:     1,
		MemoryMiB: memMiB,
		Cores:     60, // 60% cores, leave room for other workloads
		Vendor:    VendorNVIDIA,
	}
}
