package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"time"

	//"github.com/ellistarn/slang/pkg/pretty"

	//"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	//
	// Uncomment to load all auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth"
	//
	// Or uncomment to load specific auth plugins
	// _ "k8s.io/client-go/plugin/pkg/client/auth/azure"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	// _ "k8s.io/client-go/plugin/pkg/client/auth/openstack"
)

func main() {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

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
	for {
		pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		for _, p := range pods.Items {
			vols := p.Spec.Volumes
			for _, v := range vols {
				claim := v.PersistentVolumeClaim
				if claim == nil {
					continue
				}
				//   volumes:
				//   - name: inflate-sc-vol
				//     persistentVolumeClaim:
				//       claimName: inflate-sc-vol-inflate-sc-0
				claimName := claim.ClaimName
				fmt.Printf("claimName: %s\n", claimName)
				pvc, err := clientset.CoreV1().PersistentVolumeClaims(p.Namespace).Get(context.TODO(), claimName, metav1.GetOptions{})
				if err != nil {
					fmt.Println("pvc err: ", err)
					continue
				}

				if pvc.Spec.VolumeName != "" {
					// looks like the PV will have the nodeAffinity set right, just need to copy it over
					pv, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, metav1.GetOptions{})
					if err != nil {
						fmt.Println("pv err: ", err)
						continue
					}
					fmt.Printf("  pv for pod '%s' is: %s\n", p.Name, pv.Name)
				}

				scName := pvc.Spec.StorageClassName
				if scName != nil {
					fmt.Printf("  storageClassName: %s\n", *scName)
					sc, err := clientset.StorageV1().StorageClasses().Get(context.TODO(), *scName, metav1.GetOptions{})
					if err != nil {
						fmt.Println("sc err: ", err)
						continue
					}
					for _, topology := range sc.AllowedTopologies {
						for _, expr := range topology.MatchLabelExpressions {
							fmt.Printf("  match label expression: %s: %+v\n", expr.Key, expr.Values)
						}

					}
				}
			}
		}

		time.Sleep(10 * time.Second)
	}
}
