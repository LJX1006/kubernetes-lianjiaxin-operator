/*
Copyright 2021.

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

/*
my definition:
判断资源是否存在，不存在则创建，存在则更新为最新的(并没有真正更新，而是输出了一段话

*/
package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	//"k8s.io/apimachinery/third_party/forked/golang/reflect"

	examplecomv0 "example.com/v0/api/v0"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HelloWorldManagerReconciler reconciles a HelloWorldManager object
type HelloWorldManagerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log logr.Logger
}

//+kubebuilder:rbac:groups=example.com.example.com,resources=helloworldmanagers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.com.example.com,resources=helloworldmanagers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.com.example.com,resources=helloworldmanagers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HelloWorldManager object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *HelloWorldManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)
	_ = context.Background()
	_ = r.Log.WithValues("helloworldmanager",req.NamespacedName)

	// your logic here
	helloWorldManagerInstance := &examplecomv0.HelloWorldManager{}
	err  := r.Client.Get(context.TODO(),req.NamespacedName,helloWorldManagerInstance)
	r.Log.WithValues("k8s client error:",err)
	if err != nil{
		if errors.IsNotFound(err){
			return ctrl.Result{},nil
		}
		return ctrl.Result{},nil
	}
	if helloWorldManagerInstance.DeletionTimestamp != nil{
		return ctrl.Result{},nil
	}
	deployment := &appsv1.Deployment{}
	if err := r.Client.Get(context.TODO(),req.NamespacedName,deployment);err != nil && errors.IsNotFound(err){
		//cang jian guan lian zi yuan
		//1.chuang jian Deploy
		deploy := NewDeploy(helloWorldManagerInstance)
		if err := r.Client.Create(context.TODO(),deploy);err != nil{
			return ctrl.Result{},nil
		}
		//2.chuang jian Service
		service := NewService(helloWorldManagerInstance)
		if err := r.Client.Create(context.TODO(),service);err != nil{
			return ctrl.Result{},nil
		}
		//3.guan lian Annotations
		data,_ := json.Marshal(helloWorldManagerInstance.Spec)
		if helloWorldManagerInstance.Annotations != nil{
			helloWorldManagerInstance.Annotations["spec"] = string(data)
		}else{
			helloWorldManagerInstance.Annotations = map[string]string{"spec":string(data)}
		}
		if err := r.Client.Update(context.TODO(),helloWorldManagerInstance);err != nil{
			return ctrl.Result{},nil
		}
		return ctrl.Result{},nil
	}
	oldspec := &examplecomv0.HelloWorldManagerSpec{}
	fmt.Printf("--------%v------\n",helloWorldManagerInstance.Annotations["spec"])
	if err := json.Unmarshal([]byte(helloWorldManagerInstance.Annotations["spec"]),oldspec);err != nil{
		return ctrl.Result{},nil
	}
	fmt.Printf("------副本数：%v-----\n",*helloWorldManagerInstance.Spec.Size)
	if !reflect.DeepEqual(helloWorldManagerInstance.Spec,oldspec){
		//gengxin guanlian ziyuan, update deployment and service,TODO
		fmt.Printf("------%v------\n","update resources")
		return ctrl.Result{},nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelloWorldManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplecomv0.HelloWorldManager{}).
		Complete(r)
}

/*my definition*/
func NewDeploy(app *examplecomv0.HelloWorldManager) *appsv1.Deployment  {
	labels := map[string]string{"app":app.Name}
	selector := &metav1.LabelSelector{MatchLabels: labels}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind: "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: app.Name,
			Namespace: app.Namespace,

			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app,schema.GroupVersionKind{
					Group: v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind: "HelloWorldManager",
				}),
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: app.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(app),
				},
			},
		Selector: selector,
		},
	}
}
func newContainers(app *examplecomv0.HelloWorldManager)[]corev1.Container  {
	containerPorts := []corev1.ContainerPort{}
	for _,svcPort := range app.Spec.Ports{
		cport := corev1.ContainerPort{}
		cport.ContainerPort = svcPort.TargetPort.IntVal
		containerPorts = append(containerPorts,cport)
	}
	return []corev1.Container{
		{
			Name: app.Name,
			Image: app.Spec.Image,
			Resources: app.Spec.Resources,
			Ports: containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: app.Spec.Envs,
		},
	}
}
func NewService(app *examplecomv0.HelloWorldManager)*corev1.Service  {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind: "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: app.Name,
			Namespace: app.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(app,schema.GroupVersionKind{
					Group: v1.SchemeGroupVersion.Group,
					Version: v1.SchemeGroupVersion.Version,
					Kind: "HelloWorldManager",
				}),
			},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Ports: app.Spec.Ports,
			Selector: map[string]string{
				"app":app.Name,
			},
		},
	}
}