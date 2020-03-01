package virtualmachinehorizontalscaler

import (
	"context"
	"fmt"
	"math"
	"time"

	kubevmv1alpha1 "github.com/randomswdev/kubevm/pkg/apis/kubevm/v1alpha1"
	"github.com/randomswdev/kubevm/pkg/controller/virtualmachinedeployment"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/utils/gnocchi"
	"github.com/gophercloud/utils/gnocchi/metric/v1/measures"
	"github.com/gophercloud/utils/gnocchi/metric/v1/resources"
)

var log = logf.Log.WithName("controller_virtualmachinehorizontalscaler")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new VirtualMachineHorizontalScaler Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	reconciler, err := newReconciler(mgr)
	if err != nil {
		return err
	}

	return add(mgr, reconciler)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) (reconcile.Reconciler, error) {
	opts, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		return nil, err
	}

	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, err
	}

	identityClient, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return nil, err
	}

	serverClient, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return nil, err
	}

	metricClient, err := gnocchi.NewGnocchiV1(provider, gophercloud.EndpointOpts{
		Region: "RegionOne",
	})
	if err != nil {
		return nil, err
	}

	return &ReconcileVirtualMachineHorizontalScaler{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		provider: provider,
		identity: identityClient,
		metric:   metricClient,
		server:   serverClient,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("virtualmachinehorizontalscaler-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VirtualMachineHorizontalScaler
	err = c.Watch(&source.Kind{Type: &kubevmv1alpha1.VirtualMachineHorizontalScaler{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileVirtualMachineHorizontalScaler implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVirtualMachineHorizontalScaler{}

// ReconcileVirtualMachineHorizontalScaler reconciles a VirtualMachineHorizontalScaler object
type ReconcileVirtualMachineHorizontalScaler struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	// Add OpenStack clients
	provider *gophercloud.ProviderClient
	identity *gophercloud.ServiceClient
	metric   *gophercloud.ServiceClient
	server   *gophercloud.ServiceClient
}

// Reconcile reads that state of the cluster for a VirtualMachineHorizontalScaler object and makes changes based on the state read
// and what is in the VirtualMachineHorizontalScaler.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualMachineHorizontalScaler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VirtualMachineHorizontalScaler")

	// Fetch the VirtualMachineHorizontalScaler instance
	scaler := &kubevmv1alpha1.VirtualMachineHorizontalScaler{}
	err := r.client.Get(context.TODO(), request.NamespacedName, scaler)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check the VirtualMachineHorizontalScaler is valid
	if scaler.Spec.MinReplicas > scaler.Spec.MaxReplicas {
		return reconcile.Result{}, fmt.Errorf("The value of minReplicas is larger than the value of maxReplicas")
	}

	// Search for the deployment
	// Fetch the VirtualMachineDeployment instance
	deployment := &kubevmv1alpha1.VirtualMachineDeployment{}
	deploymentKey := client.ObjectKey{
		Namespace: request.Namespace,
		Name:      scaler.Spec.ScaleTargetRef.Name,
	}
	err = r.client.Get(context.TODO(), deploymentKey, deployment)
	if err != nil {
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the deployment requires an update
	deploymentUpdated := false
	if deployment.Spec.Replicas < scaler.Spec.MinReplicas {
		deployment.Spec.Replicas = scaler.Spec.MinReplicas
		deploymentUpdated = true
	} else if deployment.Spec.Replicas > scaler.Spec.MaxReplicas {
		deployment.Spec.Replicas = scaler.Spec.MaxReplicas
		deploymentUpdated = true
	} else {
		requiredReplicas, err := r.computeRequiredReplicas(scaler, deployment)
		if err != nil {
			reqLogger.Error(err, "Unable to compute Replicas value for the VirtualMachineDeployment.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return reconcile.Result{}, err
		}
		if requiredReplicas != deployment.Spec.Replicas {
			deployment.Spec.Replicas = requiredReplicas
			deploymentUpdated = true
		}
	}

	if deploymentUpdated {
		reqLogger.Info("The deployment is going to be updated.", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name, "Deployment.Spec.Replicas", deployment.Spec.Replicas)

		err = r.client.Update(context.TODO(), deployment)
		if err != nil {
			reqLogger.Error(err, "Failed to update VirtualMachineDeployment.")
			return reconcile.Result{}, err
		}

		// Delay re-evaluation for 5 minutes to let the deployment settle
		return reconcile.Result{RequeueAfter: time.Minute * 5}, nil
	}

	// Re-evaluate in 1 minute
	return reconcile.Result{RequeueAfter: time.Minute * 1}, nil
}

func (r *ReconcileVirtualMachineHorizontalScaler) computeRequiredReplicas(scaler *kubevmv1alpha1.VirtualMachineHorizontalScaler, deployment *kubevmv1alpha1.VirtualMachineDeployment) (int32, error) {
	// Get the list of VMs associated to this one
	VMList := &kubevmv1alpha1.VirtualMachineInstanceList{}
	listOpts := []client.ListOption{
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels(virtualmachinedeployment.LabelsForDeployment(deployment)),
	}
	err := r.client.List(context.TODO(), VMList, listOpts...)
	if err != nil {
		return 0, err
	}

	// Extract the array of machine IDs
	vmIDs := []string{}
	for _, vm := range VMList.Items {
		if len(vm.Status.ID) == 0 {
			return 0, fmt.Errorf("Empty VM ID")
		}
		vmIDs = append(vmIDs, vm.Status.ID)
	}

	// Find the amount of ns used by each VM in the last 60 seconds
	var ns float64 = 0
	for _, vmID := range vmIDs {
		// Extract the resource info
		resourceType := "instance"
		vm, err := resources.Get(r.metric, resourceType, vmID).Extract()
		if err != nil {
			return 0, err
		}

		// Get the CPU metric from the VM
		cpuMetricID, ok := vm.Metrics["cpu"]
		if !ok {
			return 0, fmt.Errorf("Unable to find cpu metric for %s", vmID)
		}

		// Extract the metrics frot he last 5 minutes
		startTime := time.Now().Add(time.Minute * -5).UTC()
		listOpts := measures.ListOpts{
			Refresh:     true,
			Granularity: "60s",
			Start:       &startTime,
		}
		allPages, err := measures.List(r.metric, cpuMetricID, listOpts).AllPages()
		if err != nil {
			return 0, err
		}

		allMeasures, err := measures.ExtractMeasures(allPages)
		if err != nil {
			return 0, err
		}

		mLen := len(allMeasures)
		if mLen < 2 {
			return 0, fmt.Errorf("Not enough measures for VM %s", vmID)
		}

		ns += allMeasures[mLen-1].Value - allMeasures[mLen-2].Value
	}

	requiredVMs := int32(math.Ceil((ns / float64(60*1000000000)) * float64(100) / float64(scaler.Spec.Metrics[0].Resource.Resource.AverageUtilization)))

	if requiredVMs > scaler.Spec.MaxReplicas {
		requiredVMs = scaler.Spec.MaxReplicas
	} else if requiredVMs < scaler.Spec.MinReplicas {
		requiredVMs = scaler.Spec.MinReplicas
	}

	return requiredVMs, nil
}
