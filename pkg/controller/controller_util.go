package controller

import (
	"fmt"

	batch "k8s.io/api/batch/v1"
	batchv2 "k8s.io/api/batch/v2alpha1"
	api "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	wapi "github.com/amadeusitgroup/workflow-controller/pkg/api/workflow/v1"
)

// IsWorkflowFinished checks wether a workflow is finished. A workflow is finished if one of its condition is Complete or Failed.
func IsWorkflowFinished(w *wapi.Workflow) bool {
	for _, c := range w.Status.Conditions {
		if c.Status == api.ConditionTrue && (c.Type == wapi.WorkflowComplete || c.Type == wapi.WorkflowFailed) {
			return true
		}
	}
	return false
}

// RemoveStepFromSpec remove Step from Workflow Spec
func RemoveStepFromSpec(w *wapi.Workflow, stepName string) error {
	for i := range w.Spec.Steps {
		if w.Spec.Steps[i].Name == stepName {
			w.Spec.Steps = w.Spec.Steps[:i+copy(w.Spec.Steps[i:], w.Spec.Steps[i+1:])]
			return nil
		}
	}
	return fmt.Errorf("unable to find step %q in workflow", stepName)
}

// GetStepByName returns a pointer to WorkflowStep
func GetStepByName(w *wapi.Workflow, stepName string) *wapi.WorkflowStep {
	for i := range w.Spec.Steps {
		if w.Spec.Steps[i].Name == stepName {
			return &w.Spec.Steps[i]
		}
	}
	return nil
}

// GetStepStatusByName return a pointer to WorkflowStepStatus
func GetStepStatusByName(w *wapi.Workflow, stepName string) *wapi.WorkflowStepStatus {
	for i := range w.Status.Statuses {
		if w.Status.Statuses[i].Name == stepName {
			return &w.Status.Statuses[i]
		}
	}
	return nil
}

func getJobAnnotationsSet(workflow *wapi.Workflow, template *batchv2.JobTemplateSpec) (labels.Set, error) {
	desiredAnnotations := make(labels.Set)
	for k, v := range workflow.Annotations {
		desiredAnnotations[k] = v
	}
	// TODO: add createdByRef
	return desiredAnnotations, nil // no error for the moment, when we'll add createdByRef an error could be returned
}

// TODO: implement this and DONT use ConvertSelectorToLabelsMap!
func fetchLabelsSetFromLabelSelector(selector *metav1.LabelSelector) labels.Set {
	return selector.MatchLabels
}

func getJobLabelsSet(workflow *wapi.Workflow, template *batchv2.JobTemplateSpec, stepName string) (labels.Set, error) {
	desiredLabels := fetchLabelsSetFromLabelSelector(workflow.Spec.Selector)
	for k, v := range template.Labels {
		desiredLabels[k] = v
	}
	desiredLabels[WorkflowLabelKey] = workflow.Name // add workflow name to the job  labels
	desiredLabels[WorkflowStepLabelKey] = stepName  // add stepName to the job labels
	return desiredLabels, nil
}

// createWorkflowJobLabelSelector creates label selector to select the jobs related to a workflow, stepName
func createWorkflowJobLabelSelector(workflow *wapi.Workflow, template *batchv2.JobTemplateSpec, stepName string) labels.Selector {
	set, err := getJobLabelsSet(workflow, template, stepName)
	if err != nil {
		return nil
	}
	return labels.SelectorFromSet(set)
}

func inferrWorkflowLabelSelectorForJobs(workflow *wapi.Workflow) labels.Selector {
	set := fetchLabelsSetFromLabelSelector(workflow.Spec.Selector)
	set[WorkflowLabelKey] = workflow.Name
	return labels.SelectorFromSet(set)
}

func buildOwnerReference(workflow *wapi.Workflow) metav1.OwnerReference {
	controllerRef := metav1.OwnerReference{
		APIVersion:         wapi.SchemeGroupVersion.String(),
		Kind:               wapi.ResourceKind,
		Name:               workflow.Name,
		UID:                workflow.UID,
		BlockOwnerDeletion: boolPtr(true),
		Controller:         boolPtr(true),
	}

	return controllerRef
}

func boolPtr(value bool) *bool {
	return &value
}

// IsJobFinished checks whether a Job is finished
func IsJobFinished(j *batch.Job) bool {
	for _, c := range j.Status.Conditions {
		if (c.Type == batch.JobComplete || c.Type == batch.JobFailed) && c.Status == api.ConditionTrue {
			return true
		}
	}
	return false
}

// CascadeDeleteOptions returns a DeleteOptions with Cascaded set
func CascadeDeleteOptions(gracePeriodSeconds int64) *metav1.DeleteOptions {
	return &metav1.DeleteOptions{
		GracePeriodSeconds: func(t int64) *int64 { return &t }(gracePeriodSeconds),
		PropagationPolicy: func() *metav1.DeletionPropagation {
			foreground := metav1.DeletePropagationForeground
			return &foreground
		}(),
	}
}
