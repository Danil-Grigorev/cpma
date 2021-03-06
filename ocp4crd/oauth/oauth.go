package oauth

import (
	"log"

	configv1 "github.com/openshift/api/legacyconfig/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	"gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	configv1.InstallLegacy(scheme.Scheme)
	oauthv1.Install(scheme.Scheme)
}

// TODO: Generated yamls are results of pure imagination. Structure and consistency
// must be reviewed and fixed. I guess this code can be simplified once we know
// how the output should look exactly.

// reference:
//   [v3] OCPv3:
//   - [1] https://docs.openshift.com/container-platform/3.11/install_config/configuring_authentication.html#identity_providers_master_config
//   [v4] OCPv4:
//   - [2] htpasswd: https://docs.openshift.com/container-platform/4.0/authentication/understanding-identity-provider.html
//   - [3] github: https://docs.openshift.com/container-platform/4.0/authentication/identity_providers/configuring-github-identity-provider.html

// Structures defining custom resource definitions / manifests / yamls
// TODO: figure out the OKD terminology
type identityProviderGitHub struct {
	Name          string `yaml:"name"`
	Challenge     bool   `yaml:"challenge"`
	Login         bool   `yaml:"login"`
	MappingMethod string `yaml:"mappingMethod"`
	Type          string `yaml:"type"`
	GitHub        struct {
		HostName string `yaml:"hostname"`
		CA       struct {
			Name string `yaml:"name"`
		} `yaml:"ca"`
		ClientID     string `yaml:"clientID"`
		ClientSecret struct {
			Name string `yaml:"name"`
		} `yaml:"clientSecret"`
		Organizations []string `yaml:"organizations"`
		Teams         []string `yaml:"teams"`
	} `yaml:"github"`
}

type identityProviderHTPasswd struct {
	Name          string `yaml:"name"`
	Challenge     bool   `yaml:"challenge"`
	Login         bool   `yaml:"login"`
	MappingMethod string `yaml:"mappingMethod"`
	Type          string `yaml:"type"`
	HTPasswd      struct {
		FileData struct {
			Name string `yaml:"name"`
		} `yaml:"fileData"`
	} `yaml:"htpasswd"`
}

// Shared CRD part, present in all types of OAuth CRDs
type v4OAuthCRD struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	MetaData   struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		IdentityProviders []interface{} `yaml:"identityProviders"`
	} `yaml:"spec"`
}

// Generate converts OCPv3 OAuth to OCPv4 OAuth Custom Resources
func Generate(masterconfig configv1.MasterConfig) error {
	var auth = masterconfig.OAuthConfig
	var err error

	var crd v4OAuthCRD
	crd.APIVersion = "config.openshift.io/v1"
	crd.Kind = "OAuth"
	crd.MetaData.Name = "cluster"

	serializer := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	for _, p := range auth.IdentityProviders {
		p.Provider.Object, _, err = serializer.Decode(p.Provider.Raw, nil, nil)

		switch kind := p.Provider.Object.GetObjectKind().GroupVersionKind().Kind; kind {
		case "HTPasswdPasswordIdentityProvider":
			var idP identityProviderHTPasswd
			var htpasswd configv1.HTPasswdPasswordIdentityProvider
			_, _, _ = serializer.Decode(p.Provider.Raw, nil, &htpasswd)

			idP.Type = "HTPasswd"
			idP.Challenge = p.UseAsChallenger
			idP.Login = p.UseAsLogin
			idP.MappingMethod = p.MappingMethod
			idP.HTPasswd.FileData.Name = htpasswd.File

			crd.Spec.IdentityProviders = append(crd.Spec.IdentityProviders, idP)
		case "GitHubIdentityProvider":
			var idP identityProviderGitHub
			var github configv1.GitHubIdentityProvider
			_, _, _ = serializer.Decode(p.Provider.Raw, nil, &github)

			idP.Type = "GitHub"
			idP.Name = p.Name
			idP.Challenge = p.UseAsChallenger
			idP.Login = p.UseAsLogin
			idP.MappingMethod = p.MappingMethod
			idP.GitHub.HostName = github.Hostname
			idP.GitHub.CA.Name = github.CA
			idP.GitHub.ClientID = github.ClientID
			idP.GitHub.Organizations = github.Organizations
			idP.GitHub.Teams = github.Teams
			// TODO: Learn how to handle secrets
			idP.GitHub.ClientSecret.Name = github.ClientSecret.Value

			crd.Spec.IdentityProviders = append(crd.Spec.IdentityProviders, idP)
		default:
			log.Println("can't handle: ", kind)
		}

	}

	yamlBytes, err := yaml.Marshal(&crd)
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(yamlBytes))
	return err
}
