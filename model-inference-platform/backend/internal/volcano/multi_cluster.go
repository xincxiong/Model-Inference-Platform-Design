package volcano

import (
    "context"
    "fmt"
    "sort"
    "sync"
    "time"

    "go.uber.org/zap"
    corev1 "k8s.io/api/core/v1"
    batchv1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
    volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
)

// ClusterConfig holds configuration for a single Kubernetes cluster.
type ClusterConfig struct {
    Name        string `mapstructure:"name"`
    Kubeconfig  string `mapstructure:"kubeconfig"`
    Region      string `mapstructure:"region"`
    Zone        string `mapstructure:"zone"`
    Enabled     bool   `mapstructure:"enabled"`
    IsPrimary   bool   `mapstructure:"is_primary"` // primary cluster for default routing
}

// MultiClusterClient manages Volcano scheduling across multiple Kubernetes clusters.
type MultiClusterClient struct {
    logger  *zap.Logger
    enabled bool

    mu       sync.RWMutex
    clusters map[string]*Client // cluster name → Volcano client
    primary  string             // primary cluster name

    // Resource tracking
    resourceMu   sync.RWMutex
    clusterUsage map[string]*ClusterUsage // cluster name → usage stats
}

// ClusterUsage tracks GPU/CPU/memory usage per cluster.
type ClusterUsage struct {
    ClusterName     string
    TotalGPUs       int
    AvailableGPUs   int
    TotalCPU        int64
    AvailableCPU    int64
    TotalMemoryMiB  int64
    AvailableMemMiB int64
    JobCount        int
    LastUpdated     time.Time
}

// NewMultiClusterClient creates a multi-cluster Volcano manager.
func NewMultiClusterClient(logger *zap.Logger, configs []ClusterConfig) (*MultiClusterClient, error) {
    mcc := &MultiClusterClient{
        logger:       logger,
        clusters:     make(map[string]*Client),
        clusterUsage: make(map[string]*ClusterUsage),
    }

    for _, cfg := range configs {
        if !cfg.Enabled {
            logger.Info("cluster disabled, skipping", zap.String("cluster", cfg.Name))
            continue
        }
        mcc.enabled = true

        // Build Kubernetes config
        var k8sCfg *rest.Config
        var err error

        if cfg.Kubeconfig != "" {
            k8sCfg, err = clientcmd.BuildConfigFromFlags("", cfg.Kubeconfig)
        } else {
            // Try in-cluster config
            k8sCfg, err = rest.InClusterConfig()
        }
        if err != nil {
            logger.Warn("failed to create cluster client",
                zap.String("cluster", cfg.Name),
                zap.Error(err))
            continue
        }

        volcanoClient, err := volcanoclient.NewForConfig(k8sCfg)
        if err != nil {
            logger.Warn("failed to create volcano client for cluster",
                zap.String("cluster", cfg.Name),
                zap.Error(err))
            continue
        }

        client := &Client{
            volcanoClient: volcanoClient,
            logger:        logger.With(zap.String("cluster", cfg.Name)),
            enabled:       true,
            namespace:     "inference-platform",
            clusterName:   cfg.Name,
            region:        cfg.Region,
            zone:          cfg.Zone,
        }

        mcc.clusters[cfg.Name] = client
        mcc.clusterUsage[cfg.Name] = &ClusterUsage{
            ClusterName: cfg.Name,
            LastUpdated: time.Now(),
        }

        if cfg.IsPrimary || mcc.primary == "" {
            mcc.primary = cfg.Name
        }

        logger.Info("cluster registered",
            zap.String("cluster", cfg.Name),
            zap.String("region", cfg.Region),
            zap.Bool("primary", cfg.IsPrimary))
    }

    if !mcc.enabled {
        logger.Warn("no clusters enabled, multi-cluster client will use mock mode")
    } else {
        logger.Info("multi-cluster volcano client initialized",
            zap.Int("clusters", len(mcc.clusters)),
            zap.String("primary", mcc.primary))
    }

    return mcc, nil
}

// SelectCluster chooses the best cluster for a new job based on resource availability.
func (mcc *MultiClusterClient) SelectCluster(ctx context.Context, gpuCount int32, region, zone string) (string, error) {
    if !mcc.enabled {
        return "mock-cluster", nil
    }

    mcc.mu.RLock()
    defer mcc.mu.RUnlock()

    mcc.resourceMu.RLock()
    defer mcc.resourceMu.RUnlock()

    var candidates []string
    for name, client := range mcc.clusters {
        if !client.enabled {
            continue
        }
        // Filter by region/zone if specified
        if region != "" && client.region != "" && client.region != region {
            continue
        }
        if zone != "" && client.zone != "" && client.zone != zone {
            continue
        }
        candidates = append(candidates, name)
    }

    if len(candidates) == 0 {
        // Fallback to primary
        if _, ok := mcc.clusters[mcc.primary]; ok {
            return mcc.primary, nil
        }
        return "", fmt.Errorf("no available clusters")
    }

    // Select cluster with most available GPUs
    sort.Slice(candidates, func(i, j int) bool {
        ui := mcc.clusterUsage[candidates[i]]
        uj := mcc.clusterUsage[candidates[j]]
        if ui == nil || uj == nil {
            return candidates[i] < candidates[j]
        }
        return ui.AvailableGPUs > uj.AvailableGPUs
    })

    return candidates[0], nil
}

// SubmitJobAcrossClusters submits a job to the best available cluster.
func (mcc *MultiClusterClient) SubmitJobAcrossClusters(ctx context.Context, job *batchv1alpha1.Job, gpuCount int32, region, zone string) (*batchv1alpha1.Job, string, error) {
    clusterName, err := mcc.SelectCluster(ctx, gpuCount, region, zone)
    if err != nil {
        return nil, "", err
    }

    if clusterName == "mock-cluster" {
        job.Status.State.Phase = batchv1alpha1.Running
        return job, "mock-cluster", nil
    }

    mcc.mu.RLock()
    client, ok := mcc.clusters[clusterName]
    mcc.mu.RUnlock()

    if !ok {
        return nil, "", fmt.Errorf("cluster %s not found", clusterName)
    }

    created, err := client.CreateJob(ctx, job)
    if err != nil {
        return nil, "", fmt.Errorf("submit to cluster %s: %w", clusterName, err)
    }

    // Update usage tracking
    mcc.resourceMu.Lock()
    if usage, ok := mcc.clusterUsage[clusterName]; ok {
        usage.JobCount++
        usage.AvailableGPUs -= int(gpuCount)
        usage.LastUpdated = time.Now()
    }
    mcc.resourceMu.Unlock()

    return created, clusterName, nil
}

// GetClusterUsage returns current resource usage across all clusters.
func (mcc *MultiClusterClient) GetClusterUsage(ctx context.Context) map[string]*ClusterUsage {
    mcc.resourceMu.RLock()
    defer mcc.resourceMu.RUnlock()

    result := make(map[string]*ClusterUsage)
    for k, v := range mcc.clusterUsage {
        cp := *v
        result[k] = &cp
    }
    return result
}

// RefreshClusterUsage polls each cluster for current resource state.
func (mcc *MultiClusterClient) RefreshClusterUsage(ctx context.Context) error {
    if !mcc.enabled {
        return nil
    }

    mcc.mu.RLock()
    defer mcc.mu.RUnlock()

    for name, client := range mcc.clusters {
        nodes, err := client.listNodes(ctx)
        if err != nil {
            mcc.logger.Warn("failed to list nodes for cluster",
                zap.String("cluster", name),
                zap.Error(err))
            continue
        }

        totalGPUs := 0
        availableGPUs := 0
        totalCPU := int64(0)
        availableCPU := int64(0)
        totalMemMiB := int64(0)
        availableMemMiB := int64(0)

        for _, node := range nodes {
            if node.Spec.Unschedulable {
                continue
            }

            gpuTotal := node.Status.Allocatable["nvidia.com/gpu"]
            gpuInt, _ := gpuTotal.AsInt64()
            totalGPUs += int(gpuInt)
            availableGPUs += int(gpuInt) // simplified: assume all available

            cpuTotal := node.Status.Allocatable[corev1.ResourceCPU]
            cpuInt := cpuTotal.MilliValue()
            totalCPU += cpuInt
            availableCPU += cpuInt

            memTotal := node.Status.Allocatable[corev1.ResourceMemory]
            memInt := memTotal.Value() / (1024 * 1024)
            totalMemMiB += memInt
            availableMemMiB += memInt
        }

        mcc.resourceMu.Lock()
        mcc.clusterUsage[name] = &ClusterUsage{
            ClusterName:     name,
            TotalGPUs:       totalGPUs,
            AvailableGPUs:   availableGPUs,
            TotalCPU:        totalCPU,
            AvailableCPU:    availableCPU,
            TotalMemoryMiB:  totalMemMiB,
            AvailableMemMiB: availableMemMiB,
            LastUpdated:     time.Now(),
        }
        mcc.resourceMu.Unlock()
    }

    return nil
}

// StartUsageRefresh starts periodic usage polling.
func (mcc *MultiClusterClient) StartUsageRefresh(ctx context.Context, interval time.Duration) {
    go func() {
        ticker := time.NewTicker(interval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                _ = mcc.RefreshClusterUsage(ctx)
            }
        }
    }()
}

// IsEnabled returns whether multi-cluster mode is active.
func (mcc *MultiClusterClient) IsEnabled() bool {
    return mcc.enabled
}

// PrimaryCluster returns the primary cluster name.
func (mcc *MultiClusterClient) PrimaryCluster() string {
    return mcc.primary
}

// ClusterCount returns number of registered clusters.
func (mcc *MultiClusterClient) ClusterCount() int {
    mcc.mu.RLock()
    defer mcc.mu.RUnlock()
    return len(mcc.clusters)
}
