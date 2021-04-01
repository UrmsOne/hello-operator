/*


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

package controllers

import (
	"context"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hellov1beta1 "hello-operator/api/v1beta1"
)

const helloServiceFinalizer = "hello.urmsone.com/finalizer"

// HelloServiceReconciler reconciles a HelloService object
type HelloServiceReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=hello.urmsone.com,resources=helloservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=hello.urmsone.com,resources=helloservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=hello.urmsone.com,resources=helloservices/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

func (r *HelloServiceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("helloservice", req.NamespacedName)

	// your logic here
	// Fetch the helloservice instance
	hello := &hellov1beta1.HelloService{}
	err := r.Get(ctx, req.NamespacedName, hello)
	//log.Info("hello", "replicas", hello.Spec.Replicas, "image", hello.Spec.Image)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("HelloService resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get HelloService")
		return ctrl.Result{}, err
	}

	// Check if the helloService instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	if isHelloServiceToBeDeleted := hello.GetDeletionTimestamp() != nil; isHelloServiceToBeDeleted {
		if controllerutil.ContainsFinalizer(hello, helloServiceFinalizer) {
			// Run finalization logic for helloServiceFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation
			if err := r.finalizerHelloService(log, hello); err != nil {
				// TODO:// Pre-delete callback, such release resource,
				// clean up ,close connection, and so on
				return ctrl.Result{}, err
			}
			// Remove helloServiceFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(hello, helloServiceFinalizer)
			if err := r.Update(ctx, hello); err != nil {
				r.Recorder.Eventf(hello, v1.EventTypeWarning, "InternalError", err.Error())
				return ctrl.Result{}, err
			}
		}
	} else {
		// Add finalizer for this CR
		//
		if !controllerutil.ContainsFinalizer(hello, helloServiceFinalizer) {
			controllerutil.AddFinalizer(hello, helloServiceFinalizer)
			if err := r.Update(ctx, hello); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// reconciling database deployment
	log.Info("Reconciling helloservice database")
	// coding starting database logic
	dbOpts := DatabaseOpts{}
	err = r.reconcileDatabase(dbOpts)
	if err != nil {
		log.Error(err, "Failed to reconciling helloservice database")
		return ctrl.Result{}, err
	}

	// reconciling server deployment
	r.Log.Info("Reconciling helloservice server", "apiVersion", hello.APIVersion, "helloservice", hello.Name)
	if hello.Spec.Replicas == 0 {
		hello.Spec.Replicas = 1
	}
	dOpts := ServerOpts{
		Name:      req.Name,
		Namespace: req.Namespace,
		Service:   hello,
		Scheme:    r.Scheme,
	}
	err = r.reconcileServer(dOpts)
	if err != nil {
		log.Error(err, "Failed to reconciling helloservice server", "apiVersion", hello.APIVersion)
		return ctrl.Result{}, err
	}

	// reconciling service
	r.Log.Info("Reconciling helloservice service", "helloservice", hello.Name)
	sOpts := K8sServiceOpts{
		Namespace: req.Namespace,
		Name:      req.Name,
		Service:   hello,
		Schema:    r.Scheme,
	}
	err = r.reconcileK8sService(sOpts)
	if err != nil {
		log.Error(err, "Failed to create k8s service resource for helloservice")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

type DatabaseOpts struct {
}

func (r *HelloServiceReconciler) reconcileDatabase(opts DatabaseOpts) error {
	return nil
}

func (r *HelloServiceReconciler) deploymentForServer(h *hellov1beta1.HelloService) *appv1.Deployment {
	ls := labelsForHelloService(h.Name)
	replicas := h.Spec.Replicas

	deploy := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.Name,
			Namespace: h.Namespace,
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: ls},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						ImagePullPolicy: v1.PullNever,
						Image:           h.Spec.Image,
						Name:            h.Name,
						Command:         []string{"/main"},
						Ports: []v1.ContainerPort{{
							ContainerPort: h.Spec.Port,
							Name:          "echo-server",
						}},
					}},
				},
			},
		},
	}
	ctrl.SetControllerReference(h, deploy, r.Scheme)
	return deploy
}

func labelsForHelloService(name string) map[string]string {
	return map[string]string{"app": "hello-service", "hello-service-cr": name}
}

type ServerOpts struct {
	Name      string
	Namespace string
	Service   *hellov1beta1.HelloService
	Scheme    *runtime.Scheme
}

func (r *HelloServiceReconciler) reconcileServer(opts ServerOpts) error {
	deploy := &appv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: opts.Name, Namespace: opts.Namespace}, deploy)
	if err != nil {
		if errors.IsNotFound(err) {
			deploy = deploymentForServer(opts)
			err = r.Client.Create(context.TODO(), deploy)
			if err != nil {
				r.Recorder.Eventf(deploy, v1.EventTypeWarning, "Failed to Created", err.Error())
				return err
			}
			r.Recorder.Eventf(deploy, v1.EventTypeNormal, "Created", "Successfully created helloService")
			return nil
		}
		return err
	}

	replicas := opts.Service.Spec.Replicas
	if *deploy.Spec.Replicas != replicas {
		deploy.Spec.Replicas = &replicas
		if err := r.Client.Update(context.TODO(), deploy); err != nil {
			r.Recorder.Eventf(deploy, v1.EventTypeWarning, "Failed to Updated", err.Error())
			return err
		}
	}
	return nil
}

func deploymentForServer(opts ServerOpts) *appv1.Deployment {
	ls := labelsForHelloService(opts.Name)
	deploy := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: opts.Namespace,
			Name:      opts.Name,
		},
		Spec: appv1.DeploymentSpec{
			Replicas: &opts.Service.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image:           opts.Service.Spec.Image,
						ImagePullPolicy: v1.PullNever,
						Name:            "helloservice",
						Command:         []string{"/main"},
						Ports: []v1.ContainerPort{{
							ContainerPort: opts.Service.Spec.Port,
							Name:          "helloservice",
						}},
					}},
				},
			},
		},
	}
	ctrl.SetControllerReference(opts.Service, deploy, opts.Scheme)
	return deploy
}

type K8sServiceOpts struct {
	Name      string
	Namespace string
	Service   *hellov1beta1.HelloService
	Schema    *runtime.Scheme
}

func serviceForHelloService(opts K8sServiceOpts) *v1.Service {
	ls := labelsForHelloService(opts.Name)
	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
		Spec: v1.ServiceSpec{
			//Type:     v1.ServiceTypeNodePort,
			Selector: ls,
			Ports: []v1.ServicePort{{
				Name:       opts.Name,
				Protocol:   v1.ProtocolTCP,
				Port:       opts.Service.Spec.Port,
				TargetPort: intstr.FromInt(int(opts.Service.Spec.Port)),
			}},
		},
	}
	ctrl.SetControllerReference(opts.Service, svc, opts.Schema)
	return svc
}

func (r *HelloServiceReconciler) reconcileK8sService(opts K8sServiceOpts) error {
	svc := &v1.Service{}
	ctx := context.TODO()
	err := r.Client.Get(ctx, types.NamespacedName{Name: opts.Name, Namespace: opts.Namespace}, svc)
	if err != nil {
		if errors.IsNotFound(err) {
			svc = serviceForHelloService(opts)
			err = r.Client.Create(ctx, svc)
			if err != nil {
				r.Recorder.Eventf(svc, v1.EventTypeWarning, "Failed to Created", err.Error())
				return err
			}
			r.Recorder.Eventf(svc, v1.EventTypeNormal, "Created", "Successfully created k8s Service")
			return nil
		}
		return err
	}

	return nil
}

func (r *HelloServiceReconciler) finalizerHelloService(lg logr.Logger, h *hellov1beta1.HelloService) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleteing
	// resources that are not owned by this CR, like a PVC.
	lg.Info("Successfully finalized helloService")
	return nil
}

func (r *HelloServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&hellov1beta1.HelloService{}).
		Owns(&appv1.Deployment{}).
		Owns(&v1.Service{}).
		Complete(r)
}
