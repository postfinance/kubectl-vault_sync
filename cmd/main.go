package main

import (
	"os"

	"github.com/postfinance/kubectl-vault-sync/internal/info"
	"github.com/postfinance/kubectl-vault-sync/internal/plugin"
	"github.com/spf13/pflag"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	version, commit, date string // filled by goreleaser
)

func main() {
	flags := pflag.NewFlagSet("kubectl-vault-sync", pflag.ExitOnError)
	pflag.CommandLine = flags

	v := info.New(plugin.Name, version, date, commit)
	root := plugin.NewCmdSync(genericclioptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	root.SetVersionTemplate("{{ .Version }}\n")
	root.Version = v.String()

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
