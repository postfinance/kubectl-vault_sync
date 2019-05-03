package plugin

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/postfinance/kubectl-vault-sync/pkg/job"
	"github.com/spf13/cobra"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd/api"
)

var (
	namespaceExample = `
	# synchronize all vault secrets (only works when with configured namespace annotations)
	%[1]s %[2]s

	# synchronize all vault secrets and wait for the job to finish:
	%[1]s %[2]s --wait --timeout=30s

	# synchronize a vault secret 'confidential' (only works when secretspath namespace annotation is defined)
	%[1]s %[2]s confidential

	# view batch job as yaml (this creates no batch job)
	%[1]s %[2]s --yaml

`
	longDesc = `
Synchronize vault secrets into kubernetes secrets.

This plugin creates a batch job that starts a vault-kubernetes-synchronizer container.
For more details visit https://github.com/postfinance/vault-kubernetes.

You can view the created job with: kubectl get jobs -l job=vault-sync

If you run the plugin without option, the job synchronizes all keys from vault below configured secrets path. The
configured secrets path is taken from %[1]s namespace annotation or from command line option.

Most command line options can be set with namespace annotations. For example:
	* %[2]s: configures the vault role to use for authentication
	* %[3]s: configures the mount path where the Kubernetes auth method is enabled

`
	errNoContext = fmt.Errorf("no context is currently set, use %q to select a new one", "kubectl config use-context <context>")
	dfltTimeout  = 30 * time.Second
)

const (
	// Name is the plugin name
	Name                       = "vault_sync"
	vaultSyncImageAnnotation   = "sync.vault.postfinance.ch/sync-image"
	vaultAuthImageAnnotation   = "sync.vault.postfinance.ch/auth-image"
	vaultMountpathAnnotation   = "sync.vault.postfinance.ch/mount-path"
	vaultSecretspathAnnotation = "sync.vault.postfinance.ch/secrets-path"
	vaultRoleAnnotation        = "sync.vault.postfinance.ch/role"
	vaultAddrAnnotation        = "sync.vault.postfinance.ch/addr"
	vaultTrustSecretAnnotation = "sync.vault.postfinance.ch/trust-secret"
)

const (
	dfltVaultSyncImage = "postfinance/vault-kubernetes-synchronizer:latest"
	dfltVaultAuthImage = "postfinance/vault-kubernetes-authenticator:latest"
	dfltVaultMountpath = "kubernetes"
	dfltTTL            = "1h"
)

// SyncOptions provides information required to synchronize
// vault keys as kubernetes secrets.
type SyncOptions struct {
	configFlags *genericclioptions.ConfigFlags

	userSpecifiedVaultKey         string
	userSpecifiedVaultKeyPrefix   string
	userSpecifiedVaultRole        string
	userSpecifiedVaultMountpath   string
	userSpecifiedVaultSecretsPath string
	userSpecifiedVaultSyncImage   string
	userSpecifiedVaultAuthImage   string
	userSpecifiedVaultAddr        string
	userSpecifiedVaultTrustSecret string
	userSpecifiedYAML             bool
	userSpecifiedWait             bool
	userSpecifiedTimeout          time.Duration

	rawConfig api.Config
	args      []string

	genericclioptions.IOStreams
}

// NewSyncOptions provides an instance of NamespaceOptions with default values
func NewSyncOptions(streams genericclioptions.IOStreams) *SyncOptions {
	return &SyncOptions{
		configFlags: genericclioptions.NewConfigFlags(true),

		IOStreams: streams,
	}
}

// NewCmdSync provides a cobra command wrapping SyncOptions
func NewCmdSync(streams genericclioptions.IOStreams) *cobra.Command {
	o := NewSyncOptions(streams)

	cmd := &cobra.Command{
		Long:         fmt.Sprintf(longDesc, vaultSecretspathAnnotation, vaultRoleAnnotation, vaultMountpathAnnotation),
		Example:      fmt.Sprintf(namespaceExample, "kubectl", Name),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			return o.Run()
		},
	}

	cmd.Flags().StringVar(&o.userSpecifiedVaultRole, "vault-role", "",
		fmt.Sprintf("Name of the vault role to use for authentication. If not set, value is taken from namespace annotation '%s'.", vaultRoleAnnotation))
	cmd.Flags().StringVar(&o.userSpecifiedVaultSecretsPath, "vault-secretspath", "",
		fmt.Sprintf("Secrets path in vault. If not set, value is taken from namespace annotation '%s'.", vaultSecretspathAnnotation))
	cmd.Flags().StringVar(&o.userSpecifiedVaultAddr, "vault-addr", "",
		fmt.Sprintf("The URL the vault server. If not set, value is taken from namespace annotation '%s'.", vaultAddrAnnotation))
	cmd.Flags().StringVar(&o.userSpecifiedVaultTrustSecret, "vault-trust-secret", "",
		fmt.Sprintf("The kubernetes secret containing a CA certificate 'truststore.pem' to connect to vault. If not set, value is taken from namespace annotation '%s'.", vaultTrustSecretAnnotation))
	cmd.Flags().StringVar(&o.userSpecifiedVaultSyncImage, "vault-sync-image", dfltVaultSyncImage,
		fmt.Sprintf("The synchronizer image name. If not set, value is taken from namespace annotation '%s' if it exists.", vaultSyncImageAnnotation))
	cmd.Flags().StringVar(&o.userSpecifiedVaultAuthImage, "vault-auth-image", dfltVaultAuthImage,
		fmt.Sprintf("The authorizer image name. If not set, value is taken from namespace annotation '%s' if it exists.", vaultAuthImageAnnotation))
	cmd.Flags().StringVar(&o.userSpecifiedVaultMountpath, "vault-mountpath", dfltVaultMountpath,
		fmt.Sprintf("Name of the mount path where the Kubernetes auth method is enabled. If not set, value is taken from namespace annotation '%s' if it exists.", vaultMountpathAnnotation))

	cmd.Flags().StringVar(&o.userSpecifiedVaultKeyPrefix, "vault-secret-prefix", "v3t-",
		"Prefix secrets in kubernetes. A vault secret with name 'confidential' will be synchronized in kubernetes with name '<prefix>-confidential'.")
	cmd.Flags().BoolVar(&o.userSpecifiedYAML, "yaml", false,
		"Print job yaml to stdout.")
	cmd.Flags().BoolVar(&o.userSpecifiedWait, "wait", false,
		"Wait for job to finish or fail.")
	cmd.Flags().DurationVar(&o.userSpecifiedTimeout, "timeout", dfltTimeout,
		"The length of time to wait before giving up (in combination with --wait flag).")
	o.configFlags.AddFlags(cmd.Flags())

	return cmd
}

// Complete sets all information required for updating the current context
func (o *SyncOptions) Complete(cmd *cobra.Command, args []string) error {
	o.args = args

	var err error
	o.rawConfig, err = o.configFlags.ToRawKubeConfigLoader().RawConfig()
	if err != nil {
		return err
	}

	return nil
}

// Validate ensures that all required arguments and flag values are provided
func (o *SyncOptions) Validate() error {
	if len(o.rawConfig.CurrentContext) == 0 {
		return errNoContext
	}
	if len(o.args) > 1 {
		return errors.New("only one or none argument is allowed")
	}

	return nil
}

// Run creates a kubernetes batch job that starts a sync container.
func (o *SyncOptions) Run() error {
	currentNs := o.rawConfig.Contexts[o.rawConfig.CurrentContext].Namespace
	restConfig, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}
	nsClient := clientset.CoreV1().Namespaces()
	ns, err := nsClient.Get(currentNs, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("could not create namespace api client: %s", err)
	}

	if err := o.optionsFromNamespace(ns); err != nil {
		return err
	}

	// delete completed jobs
	batchClient := clientset.BatchV1().Jobs(currentNs)
	jobs, err := batchClient.List(metav1.ListOptions{LabelSelector: fmt.Sprintf("job=%s", job.Name)})
	if err != nil {
		return fmt.Errorf("could not list batch jobs: %s", err)
	}
	deletePolicy := metav1.DeletePropagationForeground
	for _, j := range jobs.Items {
		if j.Status.Active > 0 {
			continue
		}
		if err := batchClient.Delete(j.Name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			return fmt.Errorf("could not delete batch job %s: %s", j.Name, err)
		}
	}

	secretPath := strings.TrimRight(o.userSpecifiedVaultSecretsPath, "/") + "/"
	if len(o.args) > 0 {
		secretPath = path.Join(o.userSpecifiedVaultSecretsPath, o.args[0])
	}
	ttl, _ := time.ParseDuration(dfltTTL)
	suffix := time.Now().Format("20060102-150405")
	batchJob := job.New(
		job.WithSuffix(suffix),
		job.WithTTL(ttl),
		job.WithBackoffLimit(2),
		job.WithAuthenticatorImage(o.userSpecifiedVaultAuthImage),
		job.WithSynchronizerImage(o.userSpecifiedVaultSyncImage),
		job.WithSecretPrefix(o.userSpecifiedVaultKeyPrefix),
		job.WithVaultAddr(o.userSpecifiedVaultAddr),
		job.WithVaultMountpath(o.userSpecifiedVaultMountpath),
		job.WithVaultRole(o.userSpecifiedVaultRole),
		job.WithVaultSecrets(secretPath),
		job.WithTruststore(o.userSpecifiedVaultTrustSecret),
	)
	if o.userSpecifiedYAML {
		e := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)

		err = e.Encode(batchJob, os.Stdout)
		if err != nil {
			return fmt.Errorf("failed to encode yaml: %s", err)
		}
		return nil
	}
	fmt.Printf("creating sync batch job to synchronize '%s' vault key\n", secretPath)
	_, err = batchClient.Create(batchJob)
	if err != nil {
		return fmt.Errorf("could not create batch job %s: %s", batchJob.Name, err)
	}

	if !o.userSpecifiedWait {
		return nil
	}
	watch, err := batchClient.Watch(
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job=%s,jobSuffix=%s", job.Name, suffix),
		},
	)
	if err != nil {
		return fmt.Errorf("could not watch batch job %s: %s", batchJob.Name, err)
	}

	for {
		select {
		case e := <-watch.ResultChan():
			j, ok := e.Object.(*batchv1.Job)
			if !ok {
				return errors.New("unexpected whatch type")
			}
			if j.Status.Succeeded > 0 {
				return nil
			}
			if j.Status.Failed > 0 {
				return errors.New("vault-sync job failed")
			}
		case <-time.After(o.userSpecifiedTimeout):
			return fmt.Errorf("timeout %v exceeded", o.userSpecifiedTimeout)
		}
	}
}

func (o *SyncOptions) optionsFromNamespace(ns *v1.Namespace) error {
	var ok bool
	if len(o.userSpecifiedVaultSecretsPath) == 0 {
		o.userSpecifiedVaultSecretsPath, ok = ns.GetAnnotations()[vaultSecretspathAnnotation]
		if !ok {
			return fmt.Errorf("namespace %s is not configured for vault synchronization: annotation %s not found", ns.Name, vaultSecretspathAnnotation)
		}
	}
	if len(o.userSpecifiedVaultRole) == 0 {
		o.userSpecifiedVaultRole, ok = ns.GetAnnotations()[vaultRoleAnnotation]
		if !ok {
			return fmt.Errorf("namespace %s is not configured for vault synchronization: annotation %s not found", ns.Name, vaultRoleAnnotation)
		}
	}
	if len(o.userSpecifiedVaultAddr) == 0 {
		o.userSpecifiedVaultAddr, ok = ns.GetAnnotations()[vaultAddrAnnotation]
		if !ok {
			return fmt.Errorf("namespace %s is not configured for vault synchronization: annotation %s not found", ns.Name, vaultAddrAnnotation)
		}
	}
	if len(o.userSpecifiedVaultMountpath) == 0 {
		o.userSpecifiedVaultMountpath, ok = ns.GetAnnotations()[vaultMountpathAnnotation]
		if !ok {
			return fmt.Errorf("namespace %s is not configured for vault synchronization: annotation %s not found", ns.Name, vaultMountpathAnnotation)
		}
	}

	// optional
	if len(o.userSpecifiedVaultTrustSecret) == 0 {
		o.userSpecifiedVaultTrustSecret = ns.GetAnnotations()[vaultTrustSecretAnnotation]
	}

	if o.userSpecifiedVaultSyncImage == dfltVaultSyncImage {
		img := ns.GetAnnotations()[vaultSyncImageAnnotation]
		if len(img) > 0 {
			o.userSpecifiedVaultSyncImage = img
		}
	}

	if o.userSpecifiedVaultAuthImage == dfltVaultAuthImage {
		img := ns.GetAnnotations()[vaultAuthImageAnnotation]
		if len(img) > 0 {
			o.userSpecifiedVaultAuthImage = img
		}
	}

	if o.userSpecifiedVaultMountpath == dfltVaultMountpath {
		mp := ns.GetAnnotations()[vaultMountpathAnnotation]
		if len(mp) > 0 {
			o.userSpecifiedVaultMountpath = mp
		}
	}
	return nil
}
