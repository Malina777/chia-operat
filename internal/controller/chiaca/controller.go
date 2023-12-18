/*
Copyright 2023 Chia Network Inc.
*/

package chiaca

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	k8schianetv1 "github.com/chia-network/chia-operator/api/v1"
	"github.com/chia-network/chia-operator/internal/controller/common/kube"
	"github.com/cisco-open/operator-tools/pkg/reconciler"
)

// ChiaCAReconciler reconciles a ChiaCA object
type ChiaCAReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8s.chia.net,resources=chiacas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s.chia.net,resources=chiacas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s.chia.net,resources=chiacas/finalizers,verbs=update
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile
func (r *ChiaCAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	resourceReconciler := reconciler.NewReconcilerWith(r.Client, reconciler.WithLog(log))
	log.Info(fmt.Sprintf("ChiaCAReconciler ChiaCA=%s", req.NamespacedName.String()))

	// Get the custom resource
	var ca k8schianetv1.ChiaCA
	err := r.Get(ctx, req.NamespacedName, &ca)
	if err != nil && errors.IsNotFound(err) {
		// Return here, this can happen if the CR was deleted
		return ctrl.Result{}, nil
	}
	if err != nil {
		log.Error(err, fmt.Sprintf("ChiaCAReconciler ChiaCA=%s unable to fetch ChiaCA resource", req.NamespacedName))
		return ctrl.Result{}, err
	}

	// Reconcile resources, creating them if they don't exist
	sa := r.assembleServiceAccount(ctx, ca)
	res, err := kube.ReconcileServiceAccount(ctx, resourceReconciler, sa)
	if err != nil {
		if res == nil {
			res = &reconcile.Result{}
		}
		return *res, fmt.Errorf("ChiaCAReconciler ChiaCA=%s encountered error reconciling CA generator ServiceAccount: %v", req.NamespacedName, err)
	}

	role := r.assembleRole(ctx, ca)
	res, err = kube.ReconcileRole(ctx, resourceReconciler, role)
	if err != nil {
		if res == nil {
			res = &reconcile.Result{}
		}
		return *res, fmt.Errorf("ChiaCAReconciler ChiaCA=%s encountered error reconciling CA generator Role: %v", req.NamespacedName, err)
	}

	rb := r.assembleRoleBinding(ctx, ca)
	res, err = kube.ReconcileRoleBinding(ctx, resourceReconciler, rb)
	if err != nil {
		if res == nil {
			res = &reconcile.Result{}
		}
		return *res, fmt.Errorf("ChiaCAReconciler ChiaCA=%s encountered error reconciling CA generator RoleBinding: %v", req.NamespacedName, err)
	}

	// Query CA Secret
	_, notFound, err := r.getCASecret(ctx, ca)
	if err != nil {
		log.Error(err, fmt.Sprintf("ChiaCAReconciler ChiaCA=%s unable to query for ChiaCA secret", req.NamespacedName))
		return ctrl.Result{}, err
	}
	// Create CA generating Job if Secret does not already exist
	if notFound {
		job := r.assembleJob(ctx, ca)
		res, err = kube.ReconcileJob(ctx, resourceReconciler, job)
		if err != nil {
			if res == nil {
				res = &reconcile.Result{}
			}
			return *res, fmt.Errorf("ChiaCAReconciler ChiaCA=%s encountered error reconciling CA generator Job: %v", req.NamespacedName, err)
		}

		// Loop to determine if Secret was made, set to Ready once done
		for i := 1; i <= 100; i++ {
			log.Info(fmt.Sprintf("ChiaCAReconciler ChiaCA=%s waiting for ChiaCA Job to create CA Secret, iteration %d...", req.NamespacedName.String(), i))

			_, notFound, err := r.getCASecret(ctx, ca)
			if err != nil {
				log.Error(err, fmt.Sprintf("ChiaCAReconciler ChiaCA=%s unable to query for ChiaCA secret", req.NamespacedName))
				return ctrl.Result{}, err
			}

			if !notFound {
				ca.Status.Ready = true
				err = r.Status().Update(ctx, &ca)
				if err != nil {
					log.Error(err, fmt.Sprintf("ChiaCAReconciler ChiaCA=%s unable to update ChiaCA status", req.NamespacedName))
					return ctrl.Result{}, err
				}

				break
			}

			time.Sleep(10 * time.Second)
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChiaCAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8schianetv1.ChiaCA{}).
		Complete(r)
}