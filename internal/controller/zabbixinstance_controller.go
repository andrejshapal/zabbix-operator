/*
Copyright 2026.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"maps"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	zabbixv1 "github.com/andrejshapal/zabbix-operator/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ZabbixInstanceReconciler reconciles a ZabbixInstance object
type ZabbixInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=zabbix.swisscom.com,resources=zabbixinstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=zabbix.swisscom.com,resources=zabbixinstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=zabbix.swisscom.com,resources=zabbixinstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *ZabbixInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var zabbixInstance zabbixv1.ZabbixInstance
	if err := r.Get(ctx, req.NamespacedName, &zabbixInstance); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("ZabbixInstance resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get ZabbixInstance")
		return ctrl.Result{}, err
	}

	if r.Scheme == nil {
		return ctrl.Result{}, fmt.Errorf("scheme is required to set Deployment owner references")
	}

	desiredDeployments := r.generateDeployments(&zabbixInstance)
	deployments, err := r.getDeployments(ctx, &zabbixInstance)
	if err != nil {
		log.Error(err, "Failed to get existing deployments", "error", err)
		return ctrl.Result{}, err
	}

	for i, d := range deployments {
		dd := desiredDeployments[i]
		log.Info("Reconciling deployment", "deploymentType", desiredDeployments[i].Name)

		if d == nil {
			log.Info("Deployment not found, creating a new one", "deploymentType", desiredDeployments[i].Name)
			d = &appsv1.Deployment{}
			d.Name = dd.Name
			d.Namespace = dd.Namespace
		}
		// The method makes the object state match desired state
		_, err := controllerutil.CreateOrUpdate(
			ctx,
			r.Client,
			d,
			func() error {
				d.Spec = dd.Spec

				// We need to inherit labels from cr to ensure resources are discovered by ArgoCD and other tools.
				// We also need to inherit common labels from ZabbixGlobal.
				labels := zabbixInstance.Labels
				if labels == nil {
					labels = make(map[string]string)
				}
				maps.Copy(labels, d.Labels)
				maps.Copy(labels, zabbixInstance.Spec.ZabbixGlobal.CommonLabels)
				d.Labels = labels
				d.Spec.Template.Labels = labels
				d.Spec.Selector.MatchLabels = labels

				return controllerutil.SetControllerReference(&zabbixInstance, d, r.Scheme)
			},
		)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Deployment reconciled", "deploymentType", desiredDeployments[i].Name)
	}

	return ctrl.Result{}, nil
}

func (r *ZabbixInstanceReconciler) generateDeployments(z *zabbixv1.ZabbixInstance) []*appsv1.Deployment {
	serverDeployment := r.generateDeployment("server", z.Name, z.Namespace, z.Spec.ZabbixGlobal, &z.Spec.ZabbixServer.ZabbixBase)
	webDeployment := r.generateDeployment("web", z.Name, z.Namespace, z.Spec.ZabbixGlobal, &z.Spec.ZabbixWebServer.ZabbixBase)
	agentDeployment := r.generateDeployment("agent", z.Name, z.Namespace, z.Spec.ZabbixGlobal, &z.Spec.ZabbixAgent.ZabbixBase)
	return []*appsv1.Deployment{serverDeployment, webDeployment, agentDeployment}
}

func (r *ZabbixInstanceReconciler) generateDeployment(deploymentType string, name string, ns string, g *zabbixv1.ZabbixGlobal, z *zabbixv1.ZabbixBase) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-" + deploymentType,
			Namespace: ns,
			Labels:    r.getLabels(deploymentType, name),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &[]int32{1}[0],
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": name + "-" + deploymentType,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: r.getLabels(deploymentType, name),
				},
				Spec: corev1.PodSpec{
					Containers: append([]corev1.Container{
						{
							Name:            deploymentType,
							Image:           r.getImage(g, z),
							Resources:       *z.Resources,
							Env:             z.ExtraEnv,
							VolumeMounts:    z.ExtraVolumeMounts,
							LivenessProbe:   z.LivenessProbe,
							ReadinessProbe:  z.ReadinessProbe,
							StartupProbe:    z.StartupProbe,
							ImagePullPolicy: z.ZabbixImage.PullPolicy,
						},
					}, z.ExtraContainers...),
					InitContainers:   z.ExtraInitContainers,
					Volumes:          z.ExtraVolumes,
					SecurityContext:  z.PodSecurityContext,
					NodeSelector:     z.NodeSelector,
					ImagePullSecrets: z.ZabbixImage.PullSecrets,
				},
			},
		},
	}
}

func (r *ZabbixInstanceReconciler) getImage(g *zabbixv1.ZabbixGlobal, z *zabbixv1.ZabbixBase) string {
	imageRegistry := g.ImageRegistry
	if z.ZabbixImage.Registry != "" {
		imageRegistry = z.ZabbixImage.Registry
	}
	imageRepository := g.ImageRepository
	if z.ZabbixImage.Repository != "" {
		imageRepository = z.ZabbixImage.Repository
	}
	imageTag := g.ZabbixImageTag
	if z.ZabbixImage.Tag != "" {
		imageTag = z.ZabbixImage.Tag
	}
	if z.ZabbixImage.Name != "" {
		imageRepository = imageRepository + "/" + z.ZabbixImage.Name
	}
	return imageRegistry + "/" + imageRepository + ":" + imageTag
}

func (r *ZabbixInstanceReconciler) getLabels(deploymentType string, name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       name + "-" + deploymentType,
		"app.kubernetes.io/instance":   name + "-" + deploymentType,
		"app.kubernetes.io/component":  deploymentType,
		"app.kubernetes.io/managed-by": "zabbix-operator",
		"app.kubernetes.io/part-of":    name,
	}
}

func (r *ZabbixInstanceReconciler) getDeployments(ctx context.Context, z *zabbixv1.ZabbixInstance) ([]*appsv1.Deployment, error) {
	serverDeployment, err := r.getDeployment(ctx, "server", z)
	if err != nil {
		return nil, err
	}
	webDeployment, err := r.getDeployment(ctx, "web", z)
	if err != nil {
		return nil, err
	}
	agentDeployment, err := r.getDeployment(ctx, "agent", z)
	if err != nil {
		return nil, err
	}
	return []*appsv1.Deployment{serverDeployment, webDeployment, agentDeployment}, nil
}

func (r *ZabbixInstanceReconciler) getDeployment(ctx context.Context, deploymentType string, z *zabbixv1.ZabbixInstance) (*appsv1.Deployment, error) {
	var deployment appsv1.Deployment
	namespacedName := client.ObjectKey{
		Name:      z.Name + "-" + deploymentType,
		Namespace: z.Namespace,
	}
	if err := r.Get(ctx, namespacedName, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		// Error reading the object - requeue the request.
		return nil, fmt.Errorf("failed to get deployment: %v", map[string]interface{}{"deploymentType": deploymentType, "name": z.Name, "namespace": z.Namespace, "error": err})
	}
	return &deployment, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ZabbixInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&zabbixv1.ZabbixInstance{}).
		Named("zabbixinstance").
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
