package virtualmachinedeployment

import (
	"context"
	"math/rand"
	"strconv"
	"time"

	kubevmv1alpha1 "github.com/randomswdev/kubevm/pkg/apis/kubevm/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_virtualmachinedeployment")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new VirtualMachineDeployment Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileVirtualMachineDeployment{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("virtualmachinedeployment-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VirtualMachineDeployment
	err = c.Watch(&source.Kind{Type: &kubevmv1alpha1.VirtualMachineDeployment{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Pods and requeue the owner VirtualMachineDeployment
	err = c.Watch(&source.Kind{Type: &kubevmv1alpha1.VirtualMachineInstance{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &kubevmv1alpha1.VirtualMachineDeployment{},
	})
	if err != nil {
		return err
	}

	// Initialize the random number generator
	rand.Seed(time.Now().UnixNano())

	return nil
}

// blank assignment to verify that ReconcileVirtualMachineDeployment implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVirtualMachineDeployment{}

// ReconcileVirtualMachineDeployment reconciles a VirtualMachineDeployment object
type ReconcileVirtualMachineDeployment struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a VirtualMachineDeployment object and makes changes based on the state read
// and what is in the VirtualMachineDeployment.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualMachineDeployment) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VirtualMachineDeployment")

	// Fetch the VirtualMachineDeployment instance
	deployment := &kubevmv1alpha1.VirtualMachineDeployment{}
	err := r.client.Get(context.TODO(), request.NamespacedName, deployment)
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

	// Get the list of VMs associated to this one
	VMList := &kubevmv1alpha1.VirtualMachineInstanceList{}
	listOpts := []client.ListOption{
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels(LabelsForDeployment(deployment)),
	}
	err = r.client.List(context.TODO(), VMList, listOpts...)
	if err != nil {
		reqLogger.Error(err, "Failed to list VMs.", "Deployment.Namespace", deployment.Namespace, "Memcached.Name", deployment.Name)
		return reconcile.Result{}, err
	}

	// Extract the number of non terminating VMs
	missingReplicas := deployment.Spec.Replicas
	extraVMList := []kubevmv1alpha1.VirtualMachineInstance{}
	for _, vm := range VMList.Items {
		isInstanceMarkedToBeDeleted := vm.GetDeletionTimestamp() != nil
		if !isInstanceMarkedToBeDeleted && missingReplicas > 0 {
			missingReplicas--
		} else if missingReplicas == 0 {
			extraVMList = append(extraVMList, vm)
		}
	}

	if missingReplicas > 0 {
		for missingReplicas > 0 {
			// Define a new VM object
			vm, err := r.newVMForCR(deployment)
			if err != nil {
				return reconcile.Result{}, err
			}

			reqLogger.Info("Creating a new VM", "VM.Namespace", vm.Namespace, "VM.Name", vm.Name)
			err = r.client.Create(context.TODO(), vm)
			if err != nil {
				return reconcile.Result{}, err
			}

			missingReplicas--
		}
	} else if len(extraVMList) > 0 {
		for _, vm := range extraVMList {
			reqLogger.Info("Deleting a VM", "VM.Namespace", vm.Namespace, "VM.Name", vm.Name)
			err = r.client.Delete(context.TODO(), &vm)
			if err != nil {
				return reconcile.Result{}, err
			}

			missingReplicas--
		}
	}

	// VMs already exist - don't requeue
	reqLogger.Info("Skip reconcile: VMs already exist")
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func (r *ReconcileVirtualMachineDeployment) newVMForCR(cr *kubevmv1alpha1.VirtualMachineDeployment) (*kubevmv1alpha1.VirtualMachineInstance, error) {
	// Create the VM definition
	vm := &kubevmv1alpha1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-" + strconv.Itoa(rand.Intn(1000000)),
			Namespace: cr.Namespace,
			Labels:    LabelsForDeployment(cr),
		},
		Spec: cr.Spec.Template,
	}

	// Set VirtualMachineDeployment instance as the owner and controller
	if err := controllerutil.SetControllerReference(cr, vm, r.scheme); err != nil {
		return nil, err
	}

	return vm, nil
}

// LabelsForDeployment returns the labels for selecting the resources
// belonging to the given Dployment CR name.
func LabelsForDeployment(cr *kubevmv1alpha1.VirtualMachineDeployment) map[string]string {
	result := map[string]string{"virtualmachinedeployment": cr.GetName()}

	if app, ok := cr.Labels["app"]; ok {
		result["app"] = app
	}

	return result
}
