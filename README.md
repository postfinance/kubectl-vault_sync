# vault_sync plugin

[![Release](https://img.shields.io/github/release/zbindenren/kubectl-vault_sync.svg?style=for-the-badge)](https://github.com/zbindenren/kubectl-vault_sync/releases/latest)
[![Software License](https://img.shields.io/badge/license-MIT-brightgreen.svg?style=for-the-badge)](/LICENSE.md)
[![Travis](https://img.shields.io/travis/zbindenren/kubectl-vault_sync/master.svg?style=for-the-badge)](https://travis-ci.org/zbindenren/kubectl-vault_sync)
[![Go Report Card](https://img.shields.io/badge/GOREPORT-A%2B-brightgreen.svg?style=for-the-badge)](https://goreportcard.com/report/github.com/zbindenren/kubectl-vault_sync)

[![asciicast](https://asciinema.org/a/SmeN2wnX4EfeFH4bRFjEEH6ss.svg)](https://asciinema.org/a/SmeN2wnX4EfeFH4bRFjEEH6ss)

## Concept
The `vault_sync` plugin is a k8s plugin to synchronize secrets from vault as kubernetes secrets.

It works in combination with the following projects:
* [vault-kubernetes](https://github.com/postfinance/vault-kubernetes) (Scenario 2)

It uses the following namespace annotations to create a batch job, that synchronizes secrets:

* `sync.vault.postfinance.ch/sync-image`: the synchronizer image name (default: `postfinance/vault-kubernetes-synchronizer:latest`)
* `sync.vault.postfinance.ch/auth-image`: the authorizer image name (default: `postfinance/vault-kubernetes-authenticator:latest`)
* `sync.vault.postfinance.ch/mount-path`: the name of the mount where the kubernetes auth method is enabled (default: `kubernetes`)
* `sync.vault.postfinance.ch/secrets-path`: the secrets path in vault that should be syncronized to kubernets
* `sync.vault.postfinance.ch/role`: the name of the vault role to use for authentication
* `sync.vault.postfinance.ch/addr`: the vault server's URL
* `sync.vault.postfinance.ch/trust-secret`: kubernetes secret containing a CA certificate 'truststore.pem' to connect to vault

## Usage

To sync all secrets run:
```bash
$ kubectl vault_sync
creating sync batch job to synchronize 'secret/team_linux/k8s/k8s-np/appl-zoekt-e1/' vault key
```

This creates a batch job that synchronizes the secrets. You can view the job with:

```bash
kubectl get job -l job=vault-sync
NAME                         COMPLETIONS   DURATION   AGE
vault-sync-20190412-101357   1/1           9s         103s
```

To check the logs run:

```bash
$ kubectl logs $(kubectl get pods -l job-name -o jsonpath='{.items[0].metadata.name}')
2019/04/12 08:14:12 read secret/team_linux/k8s/k8s-np/appl-zoekt-e1/gitlab from vault
2019/04/12 08:14:12 update secret gitlab from vault secret secret/team_linux/k8s/k8s-np/appl-zoekt-e1/gitlab
2019/04/12 08:14:12 secrets successfully synchronized
```



