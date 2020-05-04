// Package job creates sync jobs.
package job

import (
	"sort"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Name of the generated sync job
	Name      = "vault-sync"
	tokenDir  = "/home/vault"
	tokenPath = tokenDir + "/.vault-token"
)

// New creates a synchronize job that synchronizes secrets.
func New(options ...func(*batchv1.Job)) *batchv1.Job {
	b := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: Name,
			Labels: map[string]string{
				"job": Name,
			},
		},
		Spec: batchv1.JobSpec{
			Template: apiv1.PodTemplateSpec{
				Spec: apiv1.PodSpec{
					ServiceAccountName: "vault-auth",
					RestartPolicy:      apiv1.RestartPolicyNever,
					Volumes: []apiv1.Volume{
						{
							Name: "vault-token",
							VolumeSource: apiv1.VolumeSource{
								EmptyDir: &apiv1.EmptyDirVolumeSource{
									Medium: apiv1.StorageMediumMemory,
								},
							},
						},
					},
					InitContainers: []apiv1.Container{
						{
							Name:            "vault-auth",
							ImagePullPolicy: apiv1.PullAlways,
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "vault-token",
									MountPath: tokenDir,
								},
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "VAULT_TOKEN_PATH",
									Value: tokenPath,
								},
							},
						},
					},
					Containers: []apiv1.Container{
						{
							Name:            "vault-sync",
							ImagePullPolicy: apiv1.PullAlways,
							VolumeMounts: []apiv1.VolumeMount{
								{
									Name:      "vault-token",
									MountPath: tokenDir,
								},
							},
							Env: []apiv1.EnvVar{
								{
									Name:  "VAULT_TOKEN_PATH",
									Value: tokenPath,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, opt := range options {
		opt(b)
	}

	sortEnv(b.Spec.Template.Spec.Containers[0].Env)
	sortEnv(b.Spec.Template.Spec.InitContainers[0].Env)

	return b
}

func sortEnv(env []apiv1.EnvVar) {
	sort.Slice(env, func(i, j int) bool {
		return env[i].Name < env[j].Name
	})
}
