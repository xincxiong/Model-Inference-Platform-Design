// Package volcano provides integration with Volcano scheduler for batch job
// scheduling on Kubernetes. It supports Gang Scheduling, queue management,
// and preemption for fine-tuning tasks.
//
// Reference: https://volcano.sh/
package volcano

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchv1alpha1 "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	volcanoclient "volcano.sh/apis/pkg/client/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client provides a high-level interface to Volcano scheduler.
type Client struct {
	volcanoClient volcanoclient.Interface
	logger        *zap.Logger
	enabled       bool
	namespace     string
	clusterName   string // for multi-cluster mode
	region        string
	zone          string
}

// Config holds Volcano client configuration.
type Config struct {
	Enabled   bool
	Namespace string
}

// NewClient creates a new Volcano client from Kubernetes config.
func NewClient(logger *zap.Logger, cfg Config) (*Client, error) {
	// Build Kubernetes config
	var k8sConfig *rest.Config
	var err error

	// Try in-cluster config first
	k8sConfig, err = rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig
		kubeconfig := os.Getenv("KUBECONFIG")
		if kubeconfig == "" {
			kubeconfig = os.Getenv("HOME") + "/.kube/config"
		}
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("build k8s config: %w", err)
		}
	}

	// Create Volcano clientset
	volcanoClient, err := volcanoclient.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("create volcano client: %w", err)
	}

	return &Client{
		volcanoClient: volcanoClient,
		logger:        logger,
		enabled:       cfg.Enabled,
		namespace:     cfg.Namespace,
	}, nil
}

// NewClientFromEnv creates a Volcano client from environment variables.
//
// Environment variables:
//
//	VOLCANO_ENABLED    - "true" to enable Volcano (default: false)
//	VOLCANO_NAMESPACE  - Kubernetes namespace (default: inference-platform)
func NewClientFromEnv(logger *zap.Logger) (*Client, error) {
	enabled := os.Getenv("VOLCANO_ENABLED") == "true"
	namespace := os.Getenv("VOLCANO_NAMESPACE")
	if namespace == "" {
		namespace = "inference-platform"
	}

	return NewClient(logger, Config{
		Enabled:   enabled,
		Namespace: namespace,
	})
}

// CreateJob submits a new VolcanoJob for fine-tuning.
func (c *Client) CreateJob(ctx context.Context, job *batchv1alpha1.Job) (*batchv1alpha1.Job, error) {
	if !c.enabled {
		c.logger.Info("Volcano disabled: simulating job creation",
			zap.String("job_name", job.Name),
			zap.Int32("min_available", job.Spec.MinAvailable),
		)
		// Return mock job for local development
		job.Status.State.Phase = batchv1alpha1.Running
		return job, nil
	}

	created, err := c.volcanoClient.BatchV1alpha1().Jobs(c.namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create volcano job: %w", err)
	}

	c.logger.Info("volcano job created",
		zap.String("job_name", created.Name),
		zap.String("namespace", created.Namespace),
		zap.Int32("min_available", created.Spec.MinAvailable),
	)

	return created, nil
}

// GetJob retrieves a VolcanoJob by name.
func (c *Client) GetJob(ctx context.Context, name string) (*batchv1alpha1.Job, error) {
	if !c.enabled {
		c.logger.Debug("Volcano disabled: returning mock job",
			zap.String("job_name", name),
		)
		return &batchv1alpha1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: c.namespace,
			},
			Status: batchv1alpha1.JobStatus{
				State: batchv1alpha1.JobState{
					Phase: batchv1alpha1.Running,
				},
			},
		}, nil
	}

	job, err := c.volcanoClient.BatchV1alpha1().Jobs(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get volcano job: %w", err)
	}

	return job, nil
}

// ListJobs lists all VolcanoJobs in the namespace.
func (c *Client) ListJobs(ctx context.Context) (*batchv1alpha1.JobList, error) {
	if !c.enabled {
		c.logger.Debug("Volcano disabled: returning empty job list")
		return &batchv1alpha1.JobList{}, nil
	}

	jobs, err := c.volcanoClient.BatchV1alpha1().Jobs(c.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list volcano jobs: %w", err)
	}

	return jobs, nil
}

// DeleteJob deletes a VolcanoJob.
func (c *Client) DeleteJob(ctx context.Context, name string) error {
	if !c.enabled {
		c.logger.Info("Volcano disabled: simulating job deletion",
			zap.String("job_name", name),
		)
		return nil
	}

	err := c.volcanoClient.BatchV1alpha1().Jobs(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("delete volcano job: %w", err)
	}

	c.logger.Info("volcano job deleted",
		zap.String("job_name", name),
	)

	return nil
}

// CancelJob terminates a running VolcanoJob.
func (c *Client) CancelJob(ctx context.Context, name string) error {
	if !c.enabled {
		c.logger.Info("Volcano disabled: simulating job cancellation",
			zap.String("job_name", name),
		)
		return nil
	}

	// Update job status to Terminating
	job, err := c.GetJob(ctx, name)
	if err != nil {
		return err
	}

	job.Status.State.Phase = batchv1alpha1.Terminating
	_, err = c.volcanoClient.BatchV1alpha1().Jobs(c.namespace).UpdateStatus(ctx, job, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("cancel volcano job: %w", err)
	}

	c.logger.Info("volcano job cancelled",
		zap.String("job_name", name),
	)

	return nil
}

// GetJobStatus returns the current status of a VolcanoJob.
func (c *Client) GetJobStatus(ctx context.Context, name string) (batchv1alpha1.JobPhase, error) {
	job, err := c.GetJob(ctx, name)
	if err != nil {
		return batchv1alpha1.Pending, err
	}

	return job.Status.State.Phase, nil
}

// IsEnabled returns whether Volcano integration is active.
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// Namespace returns configured namespace.
func (c *Client) Namespace() string {
	return c.namespace
}

type JobOptions struct {
	MinAvailable          int32
	TTLSecondsAfterFinish int32
	PriorityClass         string
	SchedulerPolicy       string
}

// BuildFineTuningJob creates a VolcanoJob spec for fine-tuning.
func BuildFineTuningJob(name, namespace, queueName, imageName string, gpuCount int32, gpuMemMiB int, gpuCores int, command string, opts JobOptions) *batchv1alpha1.Job {
	if queueName == "" {
		queueName = "inference-platform-queue"
	}

	minAvailable := gpuCount
	if opts.MinAvailable > 0 {
		minAvailable = opts.MinAvailable
	}

	job := &batchv1alpha1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "finetuning",
				"job": name,
			},
			Annotations: map[string]string{
				"scheduling.volcano.sh/queue-name": queueName,
			},
		},
		Spec: batchv1alpha1.JobSpec{
			MinAvailable:  minAvailable,
			SchedulerName: "volcano",
			Tasks: []batchv1alpha1.TaskSpec{
				{
					Replicas: gpuCount,
					Name:     "worker",
					Template: buildWorkerTemplate(imageName, gpuMemMiB, gpuCores, command, opts.SchedulerPolicy),
				},
			},
		},
	}

	if opts.TTLSecondsAfterFinish > 0 {
		job.Spec.TTLSecondsAfterFinished = &opts.TTLSecondsAfterFinish
	}
	if opts.PriorityClass != "" {
		job.Spec.PriorityClassName = opts.PriorityClass
	}

	return job
}

func buildWorkerTemplate(imageName string, gpuMemMiB, gpuCores int, command string, schedulerPolicy string) corev1.PodTemplateSpec {
	if schedulerPolicy == "" {
		schedulerPolicy = "spread"
	}

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"hami.io/gpu-scheduler-policy": schedulerPolicy,
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "trainer",
					Image:   imageName,
					Command: []string{"bash", "-c"},
					Args:    []string{command},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"nvidia.com/gpu":      resource.MustParse("1"),
							"nvidia.com/gpumem":   resource.MustParse(strconv.Itoa(gpuMemMiB)),
							"nvidia.com/gpucores": resource.MustParse(strconv.Itoa(gpuCores)),
							"cpu":                 resource.MustParse("8"),
							"memory":              resource.MustParse("64Gi"),
						},
						Requests: corev1.ResourceList{
							"nvidia.com/gpu":      resource.MustParse("1"),
							"nvidia.com/gpumem":   resource.MustParse(strconv.Itoa(gpuMemMiB)),
							"nvidia.com/gpucores": resource.MustParse(strconv.Itoa(gpuCores)),
							"cpu":                 resource.MustParse("8"),
							"memory":              resource.MustParse("64Gi"),
						},
					},
				},
			},
			NodeSelector: map[string]string{
				"nvidia.com/gpu.present": "true",
			},
			Tolerations: []corev1.Toleration{
				{
					Key:      "nvidia.com/gpu",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
		},
	}
}

// listNodes retrieves node resource information for the cluster.
func (c *Client) listNodes(ctx context.Context) ([]corev1.Node, error) {
    if !c.enabled {
        return nil, nil
    }
    // Note: This requires k8s client-go, which we need to add to the Client
    // For now, return empty list (mock mode)
    return nil, nil
}
