package virtualmachineinstance

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	kubevmv1alpha1 "github.com/randomswdev/kubevm/pkg/apis/kubevm/v1alpha1"
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
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
)

var log = logf.Log.WithName("controller_virtualmachineinstance")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new VirtualMachineInstance Controller and adds it to the Manager. The Manager will set fields on the Controller
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
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: "http://192.168.50.14:5000/v3",
		Username:         "subscriber-01",
		Password:         "subscriber-01",
		DomainID:         "default",
		TenantName:       "subscriber-01",
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

	networkClient, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
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

	return &ReconcileVirtualMachineInstance{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		provider: provider,
		identity: identityClient,
		network:  networkClient,
		server:   serverClient,
	}, nil
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("virtualmachineinstance-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource VirtualMachineInstance
	err = c.Watch(&source.Kind{Type: &kubevmv1alpha1.VirtualMachineInstance{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileVirtualMachineInstance implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileVirtualMachineInstance{}

// ReconcileVirtualMachineInstance reconciles a VirtualMachineInstance object
type ReconcileVirtualMachineInstance struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme

	// Add OpneStack clients
	provider *gophercloud.ProviderClient
	identity *gophercloud.ServiceClient
	network  *gophercloud.ServiceClient
	server   *gophercloud.ServiceClient
}

// Reconcile reads that state of the cluster for a VirtualMachineInstance object and makes changes based on the state read
// and what is in the VirtualMachineInstance.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileVirtualMachineInstance) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling VirtualMachineInstance")

	// Fetch the VirtualMachineInstance instance
	instance := &kubevmv1alpha1.VirtualMachineInstance{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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

	// Check if this VM already exists
	var server *servers.Server = nil
	if len(instance.Status.ID) > 0 {
		serverInfo := servers.Get(r.server, instance.Status.ID)
		if serverInfo.Err != nil {
			return reconcile.Result{}, err
		}
		server, err = serverInfo.Extract()
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Check if the APP CR was marked to be deleted
	isInstanceMarkedToBeDeleted := instance.GetDeletionTimestamp() != nil
	if isInstanceMarkedToBeDeleted {
		if server != nil {
			err := servers.Delete(r.server, server.ID).ExtractErr()
			if err != nil {
				return reconcile.Result{}, err
			}
		}

		instance.SetFinalizers(nil)

		// Update CR
		err := r.client.Update(context.TODO(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if err := r.addFinalizer(reqLogger, instance); err != nil {
		return reconcile.Result{}, err
	}

	if server == nil {
		reqLogger.Info("Creating a new VM", "VM.Name", instance.Name)

		listOpts := networks.ListOpts{
			Name: instance.Spec.NetworkName,
		}

		allPages, err := networks.List(r.network, listOpts).AllPages()
		if err != nil {
			return reconcile.Result{}, err
		}

		allNetworks, err := networks.ExtractNetworks(allPages)
		if err != nil {
			return reconcile.Result{}, err
		}

		if len(allNetworks) != 1 {
			return reconcile.Result{}, fmt.Errorf("Unexpected number of networks")
		}

		vm := newVMForCR(instance, allNetworks[0].ID, r.server)

		server, err = servers.Create(r.server, vm).Extract()
		if err != nil {
			return reconcile.Result{}, err
		}

		instance.Status.ID = server.ID

		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update VirtualMachineInstance status.")
			return reconcile.Result{}, err
		}
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: VM already exists", "VM.Name", server.Name, "VM.ID", server.ID)
	return reconcile.Result{}, nil
}

func (r *ReconcileVirtualMachineInstance) addFinalizer(reqLogger logr.Logger, m *kubevmv1alpha1.VirtualMachineInstance) error {
	if len(m.GetFinalizers()) < 1 && m.GetDeletionTimestamp() == nil {
		reqLogger.Info("Adding Finalizer for the VirtualMachineInstance")
		m.SetFinalizers([]string{"finalizer.virtualmachineinstance.kubevm.io"})

		// Update CR
		err := r.client.Update(context.TODO(), m)
		if err != nil {
			reqLogger.Error(err, "Failed to update VirtualMachineInstance with finalizer")
			return err
		}
	}
	return nil
}

// newVMForCR returns the description of a Virtual Machine with the same name of the resource
func newVMForCR(cr *kubevmv1alpha1.VirtualMachineInstance, networkID string, server *gophercloud.ServiceClient) *servers.CreateOpts {
	createOpts := servers.CreateOpts{
		Name:       cr.Name,
		ImageName:  cr.Spec.ImageName,
		FlavorName: "m1.tiny",
		Networks: []servers.Network{
			{
				UUID: networkID,
			},
		},
		ServiceClient: server,
	}

	return &createOpts
}
