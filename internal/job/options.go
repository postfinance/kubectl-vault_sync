package job

import (
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
)

// WithAuthenticatorImage configures the init container's authenticator image.
func WithAuthenticatorImage(image string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		b.Spec.Template.Spec.InitContainers[0].Image = image
	}
}

// WithSynchronizerImage configures the container's synchronizer image.
func WithSynchronizerImage(image string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		b.Spec.Template.Spec.Containers[0].Image = image
	}
}

// WithVaultAddr configures the container's vault address environment variable.
func WithVaultAddr(addr string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		e := apiv1.EnvVar{
			Name:  "VAULT_ADDR",
			Value: addr,
		}
		b.Spec.Template.Spec.InitContainers[0].Env = append(b.Spec.Template.Spec.InitContainers[0].Env, e)
		b.Spec.Template.Spec.Containers[0].Env = append(b.Spec.Template.Spec.Containers[0].Env, e)
	}
}

// WithVaultMountpath configures the init container's vault mountpath.
func WithVaultMountpath(path string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		e := apiv1.EnvVar{
			Name:  "VAULT_AUTH_MOUNT_PATH",
			Value: path,
		}
		b.Spec.Template.Spec.InitContainers[0].Env = append(b.Spec.Template.Spec.InitContainers[0].Env, e)
	}
}

// WithVaultRole configures the init container's vault role.
func WithVaultRole(role string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		e := apiv1.EnvVar{
			Name:  "VAULT_ROLE",
			Value: role,
		}
		b.Spec.Template.Spec.InitContainers[0].Env = append(b.Spec.Template.Spec.InitContainers[0].Env, e)
	}
}

// WithVaultSecrets configures the secrets to be synchronized.
func WithVaultSecrets(secrets ...string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		e := apiv1.EnvVar{
			Name:  "VAULT_SECRETS",
			Value: strings.Join(secrets, ","),
		}
		b.Spec.Template.Spec.Containers[0].Env = append(b.Spec.Template.Spec.Containers[0].Env, e)
	}
}

// WithSecretPrefix adds prefix to all secrets.
func WithSecretPrefix(prefix string) func(*batchv1.Job) {
	if !strings.HasSuffix(prefix, "-") {
		prefix += "-"
	}

	return func(b *batchv1.Job) {
		e := apiv1.EnvVar{
			Name:  "SECRET_PREFIX",
			Value: prefix,
		}
		b.Spec.Template.Spec.Containers[0].Env = append(b.Spec.Template.Spec.Containers[0].Env, e)
	}
}

// WithTruststore configures truststore to access vault api server.
func WithTruststore(secretName string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		if secretName == "" {
			return
		}

		volume := apiv1.Volume{
			Name: "truststore",
			VolumeSource: apiv1.VolumeSource{
				Secret: &apiv1.SecretVolumeSource{
					SecretName: secretName,
					Items: []apiv1.KeyToPath{
						{
							Key:  "truststore.pem",
							Path: "truststore.pem",
						},
					},
				},
			},
		}
		mount := apiv1.VolumeMount{
			Name:      "truststore",
			MountPath: "/etc/pki/vault",
			ReadOnly:  true,
		}
		e := apiv1.EnvVar{
			Name:  "VAULT_CACERT",
			Value: "/etc/pki/vault/truststore.pem",
		}

		b.Spec.Template.Spec.Volumes = append(b.Spec.Template.Spec.Volumes, volume)
		b.Spec.Template.Spec.InitContainers[0].VolumeMounts = append(b.Spec.Template.Spec.InitContainers[0].VolumeMounts, mount)
		b.Spec.Template.Spec.Containers[0].VolumeMounts = append(b.Spec.Template.Spec.Containers[0].VolumeMounts, mount)
		b.Spec.Template.Spec.InitContainers[0].Env = append(b.Spec.Template.Spec.InitContainers[0].Env, e)
		b.Spec.Template.Spec.Containers[0].Env = append(b.Spec.Template.Spec.Containers[0].Env, e)
	}
}

// WithBackoffLimit configures the backoff limit.
func WithBackoffLimit(limit int32) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		b.Spec.BackoffLimit = int32Ptr(limit)
	}
}

// WithTTL limits the lifetime of a Job that has finished.
func WithTTL(d time.Duration) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		b.Spec.TTLSecondsAfterFinished = int32Ptr(int32(d.Seconds()))
	}
}

// WithSuffix adds a suffix to the job name and a label 'jobSuffix'
// with the suffix as value.
func WithSuffix(suffix string) func(*batchv1.Job) {
	return func(b *batchv1.Job) {
		b.Name = b.Name + "-" + suffix
		b.Labels["jobSuffix"] = suffix
	}
}

func int32Ptr(i int32) *int32 { return &i }
