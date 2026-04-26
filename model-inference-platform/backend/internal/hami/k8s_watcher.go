package hami

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

type PodEventHandler struct {
	OnAdd    func(pod *corev1.Pod)
	OnUpdate func(oldPod, newPod *corev1.Pod)
	OnDelete func(pod *corev1.Pod)
}

type K8sWatcher struct {
	clientset *kubernetes.Clientset
	stopCh    chan struct{}
	pool      interface{}
}

func NewK8sWatcher(logger interface{}, pool interface{}) *K8sWatcher {
	return &K8sWatcher{stopCh: make(chan struct{})}
}

func (w *K8sWatcher) Start(clientset *kubernetes.Clientset, namespace string) {
	w.clientset = clientset
	factory := informers.NewSharedInformerFactoryWithOptions(clientset, time.Minute, informers.WithNamespace(namespace))
	podInformer := factory.Core().V1().Pods().Informer()

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				w.handlePodAdd(pod)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod, _ := oldObj.(*corev1.Pod)
			newPod, _ := newObj.(*corev1.Pod)
			w.handlePodUpdate(oldPod, newPod)
		},
		DeleteFunc: func(obj interface{}) {
			if pod, ok := obj.(*corev1.Pod); ok {
				w.handlePodDelete(pod)
			}
		},
	})

	stop := make(chan struct{})
	go factory.Start(stop)
	cache.WaitForCacheSync(stop, podInformer.HasSynced)
}

func (w *K8sWatcher) Stop() {
	close(w.stopCh)
}

func (w *K8sWatcher) handlePodAdd(pod *corev1.Pod) {}
func (w *K8sWatcher) handlePodUpdate(oldPod, newPod *corev1.Pod) {}
func (w *K8sWatcher) handlePodDelete(pod *corev1.Pod) {}

func (w *K8sWatcher) ValidatePodGPUAllocation(podName, nodeName string) bool {
	if w.clientset == nil {
		return true
	}
	return true
}
