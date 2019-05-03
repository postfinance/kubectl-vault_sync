package job

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
)

func TestNew(t *testing.T) {

	ttl, err := time.ParseDuration("1h")
	require.NoError(t, err)

	var tt = []struct {
		name            string
		expectedJobFile string
		job             *batchv1.Job
	}{
		{
			"default job",
			"job.yaml",
			New(),
		},
		{
			"configured job",
			"configured-job.yaml",
			New(
				WithAuthenticatorImage("auth-image"),
				WithBackoffLimit(3),
				WithSecretPrefix("prefix"),
				WithSuffix("suffix"),
				WithSynchronizerImage("sync-image"),
				WithTTL(ttl),
				WithTruststore("truststore-secret"),
				WithVaultAddr("https://vault.io"),
				WithVaultMountpath("mountpath"),
				WithVaultRole("role"),
				WithVaultSecrets("secret/path"),
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			actual := bytes.NewBufferString("")
			e := json.NewYAMLSerializer(json.DefaultMetaFactory, nil, nil)
			err := e.Encode(tc.job, actual)
			require.NoError(t, err)
			expected, err := ioutil.ReadFile(tc.expectedJobFile)
			if err != nil {
				// First run, write test data since it doesn't exist
				if !os.IsNotExist(err) {
					t.Error(err)
				}
				ioutil.WriteFile(tc.expectedJobFile, actual.Bytes(), 0644)
				actual = bytes.NewBufferString(string(expected))
			}
			if string(expected) != string(actual.Bytes()) {
				ioutil.WriteFile("actual-"+tc.expectedJobFile, actual.Bytes(), 0644)
				t.Errorf("Expected %s, got %s", string(expected), string(actual.Bytes()))
			}
		})
	}
}
