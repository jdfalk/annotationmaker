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

/*
Note: this is bad code. I should fix it and make it better and I will eventually
fix it but I needed something quick and dirty to do what I want.
TODO(jdfalk): Add flag to pass in map containing all annotations
TODO(jdfalk): Optimize your loops if you can
TODO(jdfalk): set the regex to find via flag
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	//
	// Uncomment to load all auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func main() {
	var kubeconfig *string
	var namespace *string
	var dryrun *bool
	var upOpts metav1.UpdateOptions
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	namespace = flag.String("namespace", "argocd", "namespace to work on")
	dryrun = flag.Bool("dryrun", true, "Set to false to disable dry run")
	flag.Parse()

	if *dryrun {
		upOpts = metav1.UpdateOptions{DryRun: []string{metav1.DryRunAll}}
	} else {
		upOpts = metav1.UpdateOptions{}
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	secrets, err := clientset.CoreV1().Secrets(*namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Printf("There are %d secrets in the %s namespace\n", len(secrets.Items), *namespace)

	for _, u := range secrets.Items {
		if t, err := regexp.MatchString(`cluster-.*`, u.GetObjectMeta().GetName()); err != nil {
			panic(err.Error())
		} else if t {
			for key, value := range u.Data {
				if r, err := regexp.MatchString(`name$`, string(key)); err != nil {
					panic(err.Error())
				} else if r {
					project := regexp.MustCompile("(.*)-x1").FindStringSubmatch(string(value))[1]
					fmt.Printf("Now processing: %s\n", project)
					u.SetAnnotations(map[string]string{
						"managed-by": "argocd.argoproj.io",
						"project":    project})
					_, err := clientset.CoreV1().Secrets(*namespace).Update(context.TODO(), &u, upOpts)
					if err != nil {
						panic(err.Error())
					}
				}
			}
		}
	}

	// Examples for error handling:
	// - Use helper functions like e.g. errors.IsNotFound()
	// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
	// namespace := "prometheus"
	// pod := "example-xxxxx"
	// _, err = clientset.CoreV1().Pods(namespace).Get(context.TODO(), pod, metav1.GetOptions{})
	// if errors.IsNotFound(err) {
	// 	fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
	// } else if statusError, isStatus := err.(*errors.StatusError); isStatus {
	// 	fmt.Printf("Error getting pod %s in namespace %s: %v\n",
	// 		pod, namespace, statusError.ErrStatus.Message)
	// } else if err != nil {
	// 	panic(err.Error())
	// } else {
	// 	fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
	// }

}
