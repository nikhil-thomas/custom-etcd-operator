package etcdcluster

import (
	"context"
	"fmt"

	demov1alpha1 "github.com/akashshinde/etcd-operator/pkg/apis/demo/v1alpha1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_etcdcluster")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new EtcdCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEtcdCluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("etcdcluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource EtcdCluster
	err = c.Watch(&source.Kind{Type: &demov1alpha1.EtcdCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileEtcdCluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileEtcdCluster{}

// ReconcileEtcdCluster reconciles a EtcdCluster object
type ReconcileEtcdCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a EtcdCluster object and makes changes based on the state read
// and what is in the EtcdCluster.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileEtcdCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling EtcdCluster")

	// Fetch the EtcdCluster instance
	instance := &demov1alpha1.EtcdCluster{}
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

	found := &v1.Deployment{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		d := r.CreateDeployment(instance)

		err = r.client.Create(context.TODO(), d)
		if err != nil {
			reqLogger.Error(err, "Failed to create deployment")
			return reconcile.Result{}, err
		}
	} else {
		err = r.client.Get(context.TODO(), request.NamespacedName, found)
		if err != nil {
			reqLogger.Error(err, "Failed to fetch Deployment")
			return reconcile.Result{}, err
		}
		d := r.UpdateDeployment(found, instance)

		err = r.client.Update(context.TODO(), d)
		if err != nil {
			reqLogger.Error(err, "Failed to update Deployment")
			return reconcile.Result{}, err
		}
	}

	status := fmt.Sprintf("Cluster created with etcd version %s", instance.Spec.Version)
	if status != instance.Status.Status {
		instance.Status.Status = status
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update status")
			return reconcile.Result{Requeue: false}, err
		}
	}
	return reconcile.Result{}, nil
}

func (r ReconcileEtcdCluster) UpdateDeployment(d *v1.Deployment, instance *demov1alpha1.EtcdCluster) *v1.Deployment {
	d.Spec.Replicas = instance.Spec.Replica
	for i := range d.Spec.Template.Spec.Containers {
		d.Spec.Template.Spec.Containers[i].Image = "bitnami/etcd:" + instance.Spec.Version
	}
	return d
}

func (r ReconcileEtcdCluster) CreateDeployment(instance *demov1alpha1.EtcdCluster) *v1.Deployment {
	d := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: instance.Spec.Replica,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "database"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "database"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "containers",
							Image: "bitnami/etcd:" + instance.Spec.Version,
							Env: []corev1.EnvVar{
								{
									Name:  "ALLOW_NONE_AUTHENTICATION",
									Value: "yes",
								},
							},
						},
					},
				},
			},
		},
	}
	d.SetOwnerReferences([]metav1.OwnerReference{
		{
			APIVersion:         instance.APIVersion,
			Kind:               instance.Kind,
			Name:               instance.Namespace,
			UID:                instance.UID,
			Controller:         nil,
			BlockOwnerDeletion: nil,
		},
	})
	return d
}
