package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

// PodStatus represents the status of a Kubernetes pod
type PodStatus struct {
	Phase          string
	Ready          bool
	PodsReady      string // e.g., "1/1"
	ContainerState string
	Reason         string
	Message        string
}

// GetPodStatus gets the status of pods for a given Helm release
func GetPodStatus(ctx context.Context, releaseName, namespace string) (*PodStatus, error) {
	executor := NewProcessExecutor()

	// Use label selector to find pods managed by this Helm release
	cmd := Command{
		Name: "kubectl",
		Args: []string{
			"get", "pods",
			"-n", namespace,
			"-l", fmt.Sprintf("app.kubernetes.io/instance=%s", releaseName),
			"-o", "json",
		},
	}

	result, err := executor.Execute(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to get pod status: %s", result.Stderr)
	}

	var podList struct {
		Items []struct {
			Status struct {
				Phase      string `json:"phase"`
				Conditions []struct {
					Type   string `json:"type"`
					Status string `json:"status"`
					Reason string `json:"reason"`
				} `json:"conditions"`
				ContainerStatuses []struct {
					Name  string `json:"name"`
					Ready bool   `json:"ready"`
					State struct {
						Running *struct{} `json:"running"`
						Waiting *struct {
							Reason  string `json:"reason"`
							Message string `json:"message"`
						} `json:"waiting"`
						Terminated *struct {
							Reason string `json:"reason"`
						} `json:"terminated"`
					} `json:"state"`
				} `json:"containerStatuses"`
			} `json:"status"`
		} `json:"items"`
	}

	if err := json.Unmarshal([]byte(result.Stdout), &podList); err != nil {
		return nil, fmt.Errorf("failed to parse pod status: %w", err)
	}

	if len(podList.Items) == 0 {
		return &PodStatus{
			Phase:     "Unknown",
			Ready:     false,
			PodsReady: "0/0",
			Reason:    "NoPods",
			Message:   "No pods found for this release",
		}, nil
	}

	// Use the first pod (most releases have a single pod, or we show representative status)
	pod := podList.Items[0]
	status := &PodStatus{
		Phase: pod.Status.Phase,
	}

	// Check container readiness
	totalContainers := len(pod.Status.ContainerStatuses)
	readyContainers := 0
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			readyContainers++
		}

		// Determine container state
		if cs.State.Running != nil {
			status.ContainerState = "running"
		} else if cs.State.Waiting != nil {
			status.ContainerState = "waiting"
			status.Reason = cs.State.Waiting.Reason
			status.Message = cs.State.Waiting.Message
		} else if cs.State.Terminated != nil {
			status.ContainerState = "terminated"
			status.Reason = cs.State.Terminated.Reason
		}
	}

	status.PodsReady = fmt.Sprintf("%d/%d", readyContainers, totalContainers)
	status.Ready = readyContainers == totalContainers && totalContainers > 0

	// Check pod conditions for additional info
	for _, cond := range pod.Status.Conditions {
		if cond.Type == "Ready" && cond.Status != "True" && cond.Reason != "" {
			if status.Reason == "" {
				status.Reason = cond.Reason
			}
		}
	}

	return status, nil
}
