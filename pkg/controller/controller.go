package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type LoadBalancerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *LoadBalancerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var service corev1.Service
	if err := r.Get(ctx, req.NamespacedName, &service); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "unable to fetch Service")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		logger.Info("Service is not of type LoadBalancer, skipping", "service", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	deploymentName := service.Name + "-lb"
	var deployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: service.Namespace}, &deployment)

	if err != nil && apierrors.IsNotFound(err) {

		err = r.ensureConfigMap(
			ctx,
			fmt.Sprintf("%s.%s.svc", service.Name, service.Namespace),
			deploymentName, service.Namespace, service.Spec.Ports)
		if err != nil {
			panic(err)
		}

		newDeployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: service.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "v1",
						Kind:       "Service",
						Name:       service.Name,
						UID:        service.UID,
					},
				},
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"service": service.Name,
						"type":    string(corev1.ServiceTypeLoadBalancer),
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"service": service.Name,
							"type":    string(corev1.ServiceTypeLoadBalancer),
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "frpc-connector",
								Image: FRPCContainerImage,
								Command: []string{
									"/bin/frpc",
								},
								Args: []string{
									"-c",
									"/etc/frpc-conf/config.json",
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "config",
										MountPath: "/etc/frpc-conf",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "config",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: deploymentName,
										},
									},
								},
							},
						},
					},
				},
			},
		}

		if err := r.Create(ctx, newDeployment); err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", newDeployment.Namespace, "Deployment.Name", newDeployment.Name)
			return ctrl.Result{}, err
		}

		logger.Info("Created new Deployment", "Deployment.Namespace", newDeployment.Namespace, "Deployment.Name", newDeployment.Name)

		deployment = *newDeployment
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	if err := r.updateServiceStatus(ctx, &service, &deployment); err != nil {
		logger.Error(err, "Failed to update Service status")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled Service", "service", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *LoadBalancerReconciler) updateServiceStatus(ctx context.Context, service *corev1.Service, deployment *appsv1.Deployment) error {
	updatedService := service.DeepCopy()

	updatedService.Status.LoadBalancer.Ingress = []corev1.LoadBalancerIngress{
		{
			Hostname: FRPExposedHost,
		},
	}

	deploymentCondition := metav1.Condition{
		Type:               "DeploymentReady",
		Status:             metav1.ConditionUnknown,
		ObservedGeneration: service.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "Checking",
		Message:            "Checking Deployment status",
	}

	if deployment.Status.ReadyReplicas == deployment.Status.Replicas {
		deploymentCondition.Status = metav1.ConditionTrue
		deploymentCondition.Reason = "DeploymentReady"
		deploymentCondition.Message = "Deployment is ready"
	} else {
		deploymentCondition.Status = metav1.ConditionFalse
		deploymentCondition.Reason = "DeploymentNotReady"
		deploymentCondition.Message = fmt.Sprintf("Deployment is not ready: %d/%d replicas are ready", deployment.Status.ReadyReplicas, deployment.Status.Replicas)
	}

	meta.SetStatusCondition(&updatedService.Status.Conditions, deploymentCondition)

	// Update the service status
	if err := r.Status().Update(ctx, updatedService); err != nil {
		return fmt.Errorf("failed to update Service status: %w", err)
	}

	return nil
}

func (r *LoadBalancerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
