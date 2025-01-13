/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8smpetermannchv1beta1 "github.com/peschmae/self-service-operator-demo/api/v1beta1"
)

// SelfServiceNamespaceReconciler reconciles a SelfServiceNamespace object
type SelfServiceNamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=k8s.mpetermann.ch,resources=selfservicenamespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=k8s.mpetermann.ch,resources=selfservicenamespaces/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=k8s.mpetermann.ch,resources=selfservicenamespaces/finalizers,verbs=update

// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SelfServiceNamespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *SelfServiceNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// get the SelfServiceNamespace object
	var ssNamespace k8smpetermannchv1beta1.SelfServiceNamespace
	if err := r.Get(ctx, req.NamespacedName, &ssNamespace); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	err := r.checkNamespaceState(ssNamespace.Name, ctx)
	if err != nil {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	// create/update namespace
	if err := r.createOrUpdateNamespace(&ssNamespace, ctx); err != nil {
		return ctrl.Result{}, err
	}

	// create/update default network policy
	if err := r.createOrUpdateDefaultNetworkPolicies(&ssNamespace, ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *SelfServiceNamespaceReconciler) checkNamespaceState(namespace string, ctx context.Context) error {
	_ = log.FromContext(ctx)

	// check if namespace exists
	var existingNamespace corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: namespace}, &existingNamespace); err != nil {
		if client.IgnoreNotFound(err) == nil {
			//log.Log.Info("Namespace does not exist", "Namespace", namespace)
		} else {
			return fmt.Errorf("unable to fetch Namespace %v", err)
		}
	} else {
		if existingNamespace.Status.Phase == corev1.NamespaceTerminating {
			return fmt.Errorf("namespace is being terminated")
		}
	}

	return nil
}

func (r *SelfServiceNamespaceReconciler) createOrUpdateNamespace(ssNamespace *k8smpetermannchv1beta1.SelfServiceNamespace, ctx context.Context) error {
	_ = log.FromContext(ctx)

	newNamspace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ssNamespace.Name,
		},
	}

	if ssNamespace.Spec.AdditionalLabels != nil {
		newNamspace.ObjectMeta.Labels = ssNamespace.Spec.AdditionalLabels
	}

	// set the kubernetes.io/metadata.name label to the namespace name
	newNamspace.ObjectMeta.Labels["kubernetes.io/metadata.name"] = ssNamespace.Name

	if ssNamespace.Spec.AdditionalAnnotations != nil {
		newNamspace.ObjectMeta.Annotations = ssNamespace.Spec.AdditionalAnnotations
	}

	if err := controllerutil.SetControllerReference(ssNamespace, newNamspace, r.Scheme); err != nil {
		return fmt.Errorf("unable to set controller reference %v", err)
	}

	// check if namespace already exists
	var existingNamespace corev1.Namespace
	if err := r.Get(ctx, client.ObjectKey{Name: newNamspace.Name}, &existingNamespace); err != nil {
		if client.IgnoreNotFound(err) == nil {
			// namespace does not exist, create it
			log.Log.Info("Creating Namespace", "Namespace", newNamspace.Name)

			if err := r.Create(ctx, newNamspace); err != nil {
				return fmt.Errorf("unable to create Namespace %v", err)
			} else {
				log.Log.Info("Namespace created", "Namespace", newNamspace.Name)
			}
		} else {
			return fmt.Errorf("unable to fetch Namespace %v", err)
		}
	} else {
		// namespace already exists, update it

		if !reflect.DeepEqual(newNamspace.ObjectMeta.Labels, existingNamespace.ObjectMeta.Labels) || !reflect.DeepEqual(newNamspace.ObjectMeta.Annotations, existingNamespace.ObjectMeta.Annotations) {

			log.Log.Info("Label or Annotations changed, updating Namespace", "Namespace", newNamspace.Name)
			log.Log.Info("New Labels", "Labels", newNamspace.ObjectMeta.Labels)
			log.Log.Info("Existing Labels", "Labels", existingNamespace.ObjectMeta.Labels)

			existingNamespace.ObjectMeta.Labels = newNamspace.ObjectMeta.Labels
			existingNamespace.ObjectMeta.Annotations = newNamspace.ObjectMeta.Annotations

			if err := r.Update(ctx, &existingNamespace); err != nil {
				return fmt.Errorf("unable to update Namespace %v", err)
			} else {
				log.Log.Info("Namespace updated", "Namespace", newNamspace.Name)
			}
		}

	}

	return nil
}

func (r *SelfServiceNamespaceReconciler) createOrUpdateDefaultNetworkPolicies(ssNamespace *k8smpetermannchv1beta1.SelfServiceNamespace, ctx context.Context) error {
	if err := r.createOrUpdateNetPolDefaultDenyAll(ssNamespace, ctx); err != nil {
		return fmt.Errorf("unable to create default-deny-all-except-dns networkpolicy %v", err)
	}

	if err := r.createOrUpdateNetPolAllowNsInternal(ssNamespace, ctx); err != nil {
		return fmt.Errorf("unable to create default-allow-same-namespace networkpolicy %v", err)
	}

	return nil
}

func (r *SelfServiceNamespaceReconciler) createOrUpdateNetPolDefaultDenyAll(ssNamespace *k8smpetermannchv1beta1.SelfServiceNamespace, ctx context.Context) error {
	_ = log.FromContext(ctx)

	var protocolUDP = corev1.ProtocolUDP

	// create a network policy that allows all ingress and egress traffic from the same namespace
	denyAllExceptDns := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-deny-all-except-dns",
			Namespace: ssNamespace.Name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			PodSelector: metav1.LabelSelector{},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": "kube-system",
								},
							},
						},
					},
					Ports: []networkingv1.NetworkPolicyPort{
						{
							Protocol: &protocolUDP,
							Port: &intstr.IntOrString{
								Type:   intstr.Int,
								IntVal: 53,
							},
						},
					},
				},
			},
		},
	}

	return r.reconcileNetworkPolicy(ssNamespace, denyAllExceptDns, ctx)
}

func (r *SelfServiceNamespaceReconciler) createOrUpdateNetPolAllowNsInternal(ssNamespace *k8smpetermannchv1beta1.SelfServiceNamespace, ctx context.Context) error {
	_ = log.FromContext(ctx)

	// create a network policy that allows all ingress and egress traffic from the same namespace
	allowNsInternal := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default-allow-same-namespace",
			Namespace: ssNamespace.Name,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress},
			PodSelector: metav1.LabelSelector{},
			Ingress: []networkingv1.NetworkPolicyIngressRule{
				{
					From: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": ssNamespace.Name,
								},
							},
						},
					},
				},
			},
			Egress: []networkingv1.NetworkPolicyEgressRule{
				{
					To: []networkingv1.NetworkPolicyPeer{
						{
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"kubernetes.io/metadata.name": ssNamespace.Name,
								},
							},
						},
					},
				},
			},
		},
	}

	return r.reconcileNetworkPolicy(ssNamespace, allowNsInternal, ctx)
}

func (r *SelfServiceNamespaceReconciler) reconcileNetworkPolicy(ssNamespace *k8smpetermannchv1beta1.SelfServiceNamespace, netpol *networkingv1.NetworkPolicy, ctx context.Context) error {

	if err := controllerutil.SetControllerReference(ssNamespace, netpol, r.Scheme); err != nil {
		return fmt.Errorf("unable to set controller reference %v", err)
	}

	var existingNetpol networkingv1.NetworkPolicy
	if err := r.Get(ctx, client.ObjectKey{Name: netpol.Name, Namespace: ssNamespace.Name}, &existingNetpol); err != nil {
		if client.IgnoreNotFound(err) == nil {
			if err := r.Create(ctx, netpol); err != nil {
				return fmt.Errorf("unable to create NetworkPolicy %v", err)
			} else {
				log.Log.Info("NetworkPolicy created", "NetworkPolicy", netpol.Name)
			}
		}

	} else {

		// network policy already exists, update it
		if !reflect.DeepEqual(netpol.Spec, existingNetpol.Spec) {
			if err := r.Update(ctx, netpol); err != nil {
				return fmt.Errorf("unable to create NetworkPolicy %v", err)
			} else {
				log.Log.Info("NetworkPolicy updated", "NetworkPolicy", netpol.Name)
			}
		}

	}
	return nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *SelfServiceNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8smpetermannchv1beta1.SelfServiceNamespace{}).
		Owns(&corev1.Namespace{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Named("selfservicenamespace").
		Complete(r)
}
