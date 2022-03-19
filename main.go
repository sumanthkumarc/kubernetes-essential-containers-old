package main

import (
	"context"
	"flag"

	// "kubernetes-essential-containers/pkg/signals"

	api_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var (
	masterURL  string
	kubeconfig string
	label      string
)

type Event struct {
	podName   string
	namespace string
	oldStatus string
	newStatus string
	reason    string
}

// Main constructs controller dependencies and create controller object, then run it.
func main() {
	klog.InitFlags(nil)
	flag.Parse()

	// @todo validate the selector value

	// set up signals so we handle the first shutdown signal gracefully
	// @todo import signals pkg sourced from sample controller?
	// stopCh := signals.SetupSignalHandler()
	stopCh := make(chan struct{})

	// @todo make this work with service account
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	// Queue - to store the objects we are interested to process
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// Informer - to list/watch the objects and add to queue
	informer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options meta_v1.ListOptions) (runtime.Object, error) {
				options.LabelSelector = label
				return kubeClient.CoreV1().Pods(meta_v1.NamespaceAll).List(context.TODO(), options)
			},
			WatchFunc: func(options meta_v1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = label
				return kubeClient.CoreV1().Pods(meta_v1.NamespaceAll).Watch(context.TODO(), options)
			},
		},
		&api_v1.Pod{},
		0, //Skip resync
		cache.Indexers{},
	)

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(OldObj interface{}, NewObj interface{}) {

			OldObjPod := OldObj.(*api_v1.Pod)
			NewObjPod := NewObj.(*api_v1.Pod)
			var oldState api_v1.ContainerState = api_v1.ContainerState{}
			var newState api_v1.ContainerState = api_v1.ContainerState{}

			labels := NewObjPod.GetLabels()
			essentialContainer := labels[label]

			containers := make(map[string]string)
			// Hack to easily check if container exists with provided name
			for _, c := range NewObjPod.Spec.Containers {
				containers[c.Name] = c.Name
			}

			if _, exists := containers[essentialContainer]; exists {

				// @todo Support for marking multiple essential containers
				for _, c := range OldObjPod.Status.ContainerStatuses {
					if c.Name == essentialContainer {
						oldState = c.State
					}
				}

				for _, c := range NewObjPod.Status.ContainerStatuses {
					if c.Name == essentialContainer {
						newState = c.State
					}
				}

				updateEvent := Event{
					podName:   OldObjPod.GetName(),
					namespace: OldObjPod.GetNamespace(),
					oldStatus: getState(oldState),
					newStatus: getState(newState),
					reason:    getStateReason(newState),
				}
				// key, err := cache.MetaNamespaceKeyFunc(NewObj)
				if err == nil {
					queue.Add(updateEvent)
				}
			} else {
				klog.Infof("No container with name %s in pod %s. Please provide valid container name as value for %s label", essentialContainer, OldObjPod.GetName(), label)
			}

		},
	})

	controller := NewController(kubeClient, informer, queue)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running controller: %s", err.Error())
	}
}

func getState(state api_v1.ContainerState) string {

	if state.Running != nil {
		return "Running"
	} else if state.Waiting != nil {
		return "Waiting"
	} else if state.Terminated != nil {
		return "Terminated"
	} else {
		return "unknown"
	}
}

func getStateReason(state api_v1.ContainerState) string {
	if state.Running != nil {
		return ""
	} else if state.Waiting != nil {
		return state.Waiting.Reason
	} else if state.Terminated != nil {
		return state.Terminated.Reason
	} else {
		return ""
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&label, "label", "essential-container", "The label having the names of essential containers, Defaults to essential-container")
}
