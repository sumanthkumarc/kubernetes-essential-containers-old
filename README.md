# kubernetes-essential-containers
Mocks the concept of essential containers from ECS in Kubernetes


# Running

Clone this repo and run `go download` at root of repo.


```
~$:  ./kubernetes-essential-containers --kubeconfig=/Users/sumanthkumarc/.kube/config --label=essential-container
I0319 17:13:41.748775   69689 controller.go:47] Starting Foo controller
I0319 17:13:41.748894   69689 controller.go:50] Waiting for informer caches to sync
I0319 17:13:41.849276   69689 controller.go:55] Starting workers
I0319 17:13:41.849306   69689 controller.go:61] Started workers



```

# Todo
1. Make this run with service account.
2. Github actions with automated image build on new release.
3. Helm chart based deployment option

