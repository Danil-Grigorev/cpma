package oauth_test

import (
	"io/ioutil"
	"testing"

	"github.com/fusor/cpma/pkg/transform/oauth"
	cpmatest "github.com/fusor/cpma/pkg/utils/test"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func TestTransformMasterConfigTemplates(t *testing.T) {
	_, templates, err := cpmatest.LoadIPTestData("testdata/templates/master_config.yaml")
	require.NoError(t, err)

	expectedContent, err := ioutil.ReadFile("testdata/templates/expected-CR-oauth.yaml")
	require.NoError(t, err)

	var expectedCrd configv1.OAuth
	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	_, _, err = serializer.Decode(expectedContent, nil, &expectedCrd)
	require.NoError(t, err)

	testCases := []struct {
		name        string
		expectedCrd *configv1.OAuth
	}{
		{
			name:        "build oauth templates",
			expectedCrd: &expectedCrd,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oauthResources, err := oauth.Translate([]oauth.IdentityProvider{}, oauth.TokenConfig{}, *templates)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedCrd, oauthResources.OAuthCRD)
		})
	}
}
