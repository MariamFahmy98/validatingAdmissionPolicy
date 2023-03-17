package main

import (
	"context"
	"fmt"
	"os"
	"strings"

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
	"k8s.io/apiserver/pkg/admission/plugin/webhook/matchconditions"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
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

	for _, resource := range resources {
		policyDecisions := applyPolicyToResource(&policy, resource)

		for _, decision := range policyDecisions {
			if strings.Compare(string(decision.Action), "deny") == 0 {
				fmt.Println(decision.Message)
			} else {
				fmt.Println(decision.Action)
			}
		}
	}
}

func applyPolicyToResource(policy *v1alpha1.ValidatingAdmissionPolicy, resource *unstructured.Unstructured) []validatingadmissionpolicy.PolicyDecision {
	forbiddenReason := metav1.StatusReasonForbidden
	matchPolicyType := v1alpha1.Exact

	var validations []v1alpha1.Validation = policy.Spec.Validations
	var expressions, messageExpressions []cel.ExpressionAccessor

	for _, expression := range validations {
		message := fmt.Sprintf("error: failed to create %s: %s \"%s\" is forbidden: ValidatingAdmissionPolicy '%s' denied request: failed expression: %s", resource.GetKind(), resource.GetAPIVersion(), resource.GetName(), policy.Name, expression.Expression)
		condition := &validatingadmissionpolicy.ValidationCondition{
			Expression: expression.Expression,
			Message:    message,
			Reason:     &forbiddenReason,
		}

		messageCondition := &validatingadmissionpolicy.MessageExpressionCondition{
			MessageExpression: expression.MessageExpression,
		}

		expressions = append(expressions, condition)
		messageExpressions = append(messageExpressions, messageCondition)
	}

	filterCompiler := cel.NewFilterCompiler()
	filter := filterCompiler.Compile(expressions, cel.OptionalVariableDeclarations{HasParams: false, HasAuthorizer: false}, celconfig.PerCallLimit)

	compileErrors := filter.CompilationErrors()

	if len(compileErrors) > 0 {
		for _, err := range compileErrors {
			fmt.Println(err.Error())
		}
		return nil
	}

	messageExpressionCompiler := cel.NewFilterCompiler()
	messageExpressionfilter := messageExpressionCompiler.Compile(messageExpressions, cel.OptionalVariableDeclarations{HasParams: false, HasAuthorizer: false}, celconfig.PerCallLimit)

	admissionAttributes := admission.NewAttributesRecord(resource.DeepCopyObject(), nil, resource.GroupVersionKind(), resource.GetNamespace(), resource.GetName(), schema.GroupVersionResource{}, "", admission.Create, nil, false, nil)
	versionedAttr, _ := admission.NewVersionedAttributes(admissionAttributes, admissionAttributes.GetKind(), nil)

	ctx := context.TODO()
	failPolicy := v1.FailurePolicyType(*policy.Spec.FailurePolicy)

	matchConditions := policy.Spec.MatchConditions
	var matchExpressions []cel.ExpressionAccessor

	for _, expression := range matchConditions {
		condition := &matchconditions.MatchCondition{
			Name:       expression.Name,
			Expression: expression.Expression,
		}
		matchExpressions = append(matchExpressions, condition)
	}

	matchFilterCompiler := cel.NewFilterCompiler()
	matchFilter := matchFilterCompiler.Compile(matchExpressions, cel.OptionalVariableDeclarations{HasParams: false, HasAuthorizer: false}, celconfig.PerCallLimit)

	newMatcher := matchconditions.NewMatcher(matchFilter, nil, &failPolicy, string(matchPolicyType), "test")

	auditAnnotations := policy.Spec.AuditAnnotations
	var auditExpressions []cel.ExpressionAccessor

	for _, expression := range auditAnnotations {
		condition := &validatingadmissionpolicy.AuditAnnotationCondition{
			Key:             expression.Key,
			ValueExpression: expression.ValueExpression,
		}
		auditExpressions = append(auditExpressions, condition)
	}

	auditAnnotationFilterCompiler := cel.NewFilterCompiler()
	auditAnnotationFilter := auditAnnotationFilterCompiler.Compile(auditExpressions, cel.OptionalVariableDeclarations{HasParams: false, HasAuthorizer: false}, celconfig.PerCallLimit)

	validator := validatingadmissionpolicy.NewValidator(filter, newMatcher, auditAnnotationFilter, messageExpressionfilter, &failPolicy, nil)
	validateResult := validator.Validate(ctx, versionedAttr, nil, celconfig.RuntimeCELCostBudget)

	//fmt.Println(validateResult.AuditAnnotations[0].Action)

	return validateResult.Decisions
}
