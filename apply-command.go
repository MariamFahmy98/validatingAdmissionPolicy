package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ApplyCommandConfig struct {
	PolicyPath   string
	ResourcePath string
}

var (
	applyHelp = `
To apply a policy on a resource:
		cobra-cli apply /path/to/policy.yaml /path/to/resource.yaml
`
)

func ApplyCommand() *cobra.Command {
	var cmd *cobra.Command
	applyCommandConfig := &ApplyCommandConfig{}

	cmd = &cobra.Command{
		Use:     "apply",
		Short:   "Applies policies on resources.",
		Example: applyHelp,
		Run: func(cmd *cobra.Command, arguments []string) {
			applyCommandConfig.PolicyPath = arguments[0]
			applyCommandConfig.ResourcePath = arguments[1]

			applyCommandConfig.applyCommandHelper()
		},
	}

	return cmd
}

func (c *ApplyCommandConfig) applyCommandHelper() {
	resourceBytes, error := os.ReadFile(c.ResourcePath)
	if error != nil {
		fmt.Println("unable to read resources file")
		return
	}

	resources, error := GetResource(resourceBytes)
	if error != nil {
		fmt.Println("unable to get resources")
		return
	}

	fmt.Println("len(resources):", len(resources))
	fmt.Println("1st resource kind:", resources[0].GetKind())
	fmt.Println("1st resource labels:", resources[0].GetLabels())
	fmt.Println("1st resource kind:", resources[0].GetObjectKind().GroupVersionKind().Kind)
	fmt.Println("1st resource labels:", resources[0].Object["spec"])
	fmt.Println("1st resource unstructured content:", resources[0].UnstructuredContent())

	var specField map[string]interface{}
	specField, _, _ = unstructured.NestedMap(resources[0].UnstructuredContent(), "spec")

	fmt.Println("spec.replicas:", specField["replicas"])

	var strategyField map[string]interface{}
	strategyField, _, _ = unstructured.NestedMap(specField, "strategy")

	fmt.Println("spec.strategy.type:", strategyField["type"])

	// userHomeDir, err := os.UserHomeDir()
	// if err != nil {
	// 	fmt.Printf("error getting user home dir: %v\n", err)
	// 	os.Exit(1)
	// }

	// kubeConfigPath := filepath.Join(userHomeDir, ".kube", "config")

	// kubeConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	// if err != nil {
	// 	fmt.Printf("error getting Kubernetes config: %v\n", err)
	// 	os.Exit(1)
	// }

	// clientset, err := kubernetes.NewForConfig(kubeConfig)
	// if err != nil {
	// 	fmt.Printf("error getting Kubernetes clientset: %v\n", err)
	// 	os.Exit(1)
	// }

	// dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	// if err != nil {
	// 	fmt.Printf("error creating dynamic client: %v\n", err)
	// 	os.Exit(1)
	// }

	// b, err := os.ReadFile(c.ResourcePath)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// log.Printf("%q \n", string(b))

	// decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewReader(b), 100)
	// for {
	// 	var rawObj runtime.RawExtension
	// 	if err = decoder.Decode(&rawObj); err != nil {
	// 		break
	// 	}

	// 	obj, gvk, _ := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme).Decode(rawObj.Raw, nil, nil)
	// 	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}

	// 	gr, err := restmapper.GetAPIGroupResources(clientset.Discovery())
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	mapper := restmapper.NewDiscoveryRESTMapper(gr)
	// 	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	var dri dynamic.ResourceInterface
	// 	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
	// 		if unstructuredObj.GetNamespace() == "" {
	// 			unstructuredObj.SetNamespace("default")
	// 		}
	// 		dri = dynamicClient.Resource(mapping.Resource).Namespace(unstructuredObj.GetNamespace())
	// 	} else {
	// 		dri = dynamicClient.Resource(mapping.Resource)
	// 	}

	// 	if _, err := dri.Create(context.Background(), unstructuredObj, metav1.CreateOptions{}); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	// if err != io.EOF {
	// 	log.Fatal("eof ", err)
	// }
}
