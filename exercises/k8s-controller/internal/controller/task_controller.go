/*
Copyright 2025.

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
	"os/exec"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/go-logr/logr"
	taskv1 "github.com/kumarlokesh/sysd/exercises/k8s-controller/api/v1"
)

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=task.task.sysd.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=task.task.sysd.io,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=task.task.sysd.io,resources=tasks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("task", req.NamespacedName)
	log.Info("Reconciling Task")

	// Create a logger with request context
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling Task")

	// Fetch the Task instance
	task := &taskv1.Task{}
	if err := r.Get(ctx, req.NamespacedName, task); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Task resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Task")
		return ctrl.Result{}, err
	}

	log.Info("Processing Task", "command", task.Spec.Command, "args", task.Spec.Args, "schedule", task.Spec.Schedule)

	// If the task has a schedule, check if it's time to run
	requeueAfter := time.Duration(0)
	if task.Spec.Schedule != "" {
		// For simplicity, we'll just log that we would schedule this task
		// In a real implementation, you would use a cron library to parse the schedule
		log.Info("Task has a schedule, will run at the specified time", "schedule", task.Spec.Schedule)
		// For now, we'll just run the task immediately for demonstration
		requeueAfter = time.Minute * 5 // Requeue after 5 minutes for demonstration
	} else {
		log.Info("Task has no schedule, executing immediately")
	}

	// Execute the command
	log.Info("Executing command", "command", task.Spec.Command, "args", task.Spec.Args)
	output, err := r.executeCommand(task.Spec.Command, task.Spec.Args...)

	log.Info("Command execution result", "output", output, "error", err)

	// Create a copy of the task to update status
	taskCopy := task.DeepCopy()

	// Update status
	now := metav1.Now()
	taskCopy.Status.LastExecutionTime = &now
	if taskCopy.Status.ExecutionCount == 0 {
		taskCopy.Status.ExecutionCount = 1
	} else {
		taskCopy.Status.ExecutionCount++
	}

	if err != nil {
		taskCopy.Status.LastError = err.Error()
		taskCopy.Status.LastExecutionOutput = ""
		log.Error(err, "error executing command")
	} else {
		taskCopy.Status.LastError = ""
		taskCopy.Status.LastExecutionOutput = output
	}

	log.Info("Updating Task status", "status", taskCopy.Status)

	// Update the Task status
	if err := r.Status().Update(ctx, taskCopy); err != nil {
		log.Error(err, "unable to update Task status")
		return ctrl.Result{}, err
	}

	log.Info("Successfully updated Task status")

	// If we have a requeue time, return it
	if requeueAfter > 0 {
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	return ctrl.Result{}, nil
}

// executeCommand executes the given command with arguments
func (r *TaskReconciler) executeCommand(command string, args ...string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("no command specified")
	}

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize logger
	r.Log = zap.New(zap.UseDevMode(true))

	// Create a new controller
	r.Log.Info("Setting up controller with manager")

	// Build the controller
	return ctrl.NewControllerManagedBy(mgr).
		For(&taskv1.Task{}).
		Complete(r)
}
