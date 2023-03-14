package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/api/admissionregistration/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/plugin/cel"
	"k8s.io/apiserver/pkg/admission/plugin/validatingadmissionpolicy"
	"k8s.io/apiserver/pkg/admission/plugin/webhook/generic"
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
	// --------------------------------
	policyBytes, error := os.ReadFile(c.PolicyPath)
	if error != nil {
		fmt.Println("unable to read policy file")
		return
	}

	policies, error := GetResource(policyBytes)
	if error != nil {
		fmt.Println("unable to get policies")
		return
	}

	var policy v1alpha1.ValidatingAdmissionPolicy
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(policies[0].Object, &policy)
	if err != nil {
		return
	}
	fmt.Println("policy.spec.validation[0].expression:", policy.Spec.Validations[0].Expression)
	fmt.Println("policy.spec.failurepolicy:", policy.Spec.FailurePolicy)
	fmt.Println("policy.Name:", policy.Name)
	// --------------------------------

	forbiddenReason := metav1.StatusReasonForbidden

	validationCondition := &validatingadmissionpolicy.ValidationCondition{
		Expression: policy.Spec.Validations[0].Expression,
		Message:    "this is the validating condition",
		Reason:     &forbiddenReason,
	}

	var expressions []cel.ExpressionAccessor
	expressions = append(expressions, validationCondition)

	filterCompiler := cel.NewFilterCompiler()

	filter := filterCompiler.Compile(expressions, false)

	// It works good
	// compileErrors := filter.CompilationErrors()
	// fmt.Println("error: ", compileErrors[0].Error())

	admissionAttributes := admission.NewAttributesRecord(resources[0].DeepCopyObject(), nil, resources[0].GroupVersionKind(), "default", "nginx", schema.GroupVersionResource{}, "", admission.Create, nil, false, nil)
	versionedAttr, _ := generic.NewVersionedAttributes(admissionAttributes, admissionAttributes.GetKind(), nil)
	fail := v1.Fail

	validator := validatingadmissionpolicy.NewValidator(filter, &fail)
	policyResults := validator.Validate(versionedAttr, nil)

	fmt.Println("policyResults[0].Action:", policyResults[0].Action)
	fmt.Println("policyResults[0].Message:", policyResults[0].Message)
	fmt.Println("policyResults[0].Reason:", policyResults[0].Reason)

}
