/*
Copyright 2019 The Kubernetes Authors.

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

package client

import (
	clusterctlv1 "sigs.k8s.io/cluster-api/cmd/clusterctl/api/v1alpha3"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/cluster"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/config"
	"sigs.k8s.io/cluster-api/cmd/clusterctl/client/repository"
)

// Client is exposes the clusterctl high-level client library.
type Client interface {
	// GetProvidersConfig returns the list of providers configured for this instance of clusterctl.
	GetProvidersConfig() ([]Provider, error)

	// GetProviderComponents returns the provider components for a given provider with options including targetNamespace, watchingNamespace.
	GetProviderComponents(provider string, providerType clusterctlv1.ProviderType, options ComponentsOptions) (Components, error)

	// Init initializes a management cluster by adding the requested list of providers.
	Init(options InitOptions) ([]Components, error)

	// InitImages returns the list of images required for executing the init command.
	InitImages(options InitOptions) ([]string, error)

	// GetClusterTemplate returns a workload cluster template.
	GetClusterTemplate(options GetClusterTemplateOptions) (Template, error)

	// GetKubeconfig returns the kubeconfig of the workload cluster.
	GetKubeconfig(options GetKubeconfigOptions) (string, error)

	// Delete deletes providers from a management cluster.
	Delete(options DeleteOptions) error

	// Move moves all the Cluster API objects existing in a namespace (or from all the namespaces if empty) to a target management cluster.
	Move(options MoveOptions) error

	// PlanUpgrade returns a set of suggested Upgrade plans for the cluster, and more specifically:
	// - Each management group gets separated upgrade plans.
	// - For each management group, an upgrade plan is generated for each API Version of Cluster API (contract) available, e.g.
	//   - Upgrade to the latest version in the the v1alpha2 series: ....
	//   - Upgrade to the latest version in the the v1alpha3 series: ....
	PlanUpgrade(options PlanUpgradeOptions) ([]UpgradePlan, error)

	// PlanCertManagerUpgrade returns a CertManagerUpgradePlan.
	PlanCertManagerUpgrade(options PlanUpgradeOptions) (CertManagerUpgradePlan, error)

	// ApplyUpgrade executes an upgrade plan.
	ApplyUpgrade(options ApplyUpgradeOptions) error

	// ProcessYAML provides a direct way to process a yaml and inspect its
	// variables.
	ProcessYAML(options ProcessYAMLOptions) (YamlPrinter, error)
}

// YamlPrinter exposes methods that prints the processed template and
// variables.
type YamlPrinter interface {
	// Variables required by the template.
	Variables() []string

	// Yaml returns yaml defining all the cluster template objects as a byte array.
	Yaml() ([]byte, error)
}

// clusterctlClient implements Client.
type clusterctlClient struct {
	configClient            config.Client
	repositoryClientFactory RepositoryClientFactory
	clusterClientFactory    ClusterClientFactory
}

// RepositoryClientFactoryInput represents the inputs required by the
// RepositoryClientFactory
type RepositoryClientFactoryInput struct {
	provider  Provider
	processor Processor
}
type RepositoryClientFactory func(RepositoryClientFactoryInput) (repository.Client, error)

// ClusterClientFactoryInput reporesents the inputs required by the
// ClusterClientFactory
type ClusterClientFactoryInput struct {
	kubeconfig Kubeconfig
	processor  Processor
}
type ClusterClientFactory func(ClusterClientFactoryInput) (cluster.Client, error)

// Ensure clusterctlClient implements Client.
var _ Client = &clusterctlClient{}

// Option is a configuration option supplied to New
type Option func(*clusterctlClient)

// InjectConfig allows to override the default configuration client used by clusterctl.
func InjectConfig(config config.Client) Option {
	return func(c *clusterctlClient) {
		c.configClient = config
	}
}

// InjectRepositoryFactory allows to override the default factory used for creating
// RepositoryClient objects.
func InjectRepositoryFactory(factory RepositoryClientFactory) Option {
	return func(c *clusterctlClient) {
		c.repositoryClientFactory = factory
	}
}

// InjectClusterClientFactory allows to override the default factory used for creating
// ClusterClient objects.
func InjectClusterClientFactory(factory ClusterClientFactory) Option {
	return func(c *clusterctlClient) {
		c.clusterClientFactory = factory
	}
}

// New returns a configClient.
func New(path string, options ...Option) (Client, error) {
	return newClusterctlClient(path, options...)
}

func newClusterctlClient(path string, options ...Option) (*clusterctlClient, error) {
	client := &clusterctlClient{}
	for _, o := range options {
		o(client)
	}

	// if there is an injected config, use it, otherwise use the default one
	// provided by the config low level library.
	if client.configClient == nil {
		c, err := config.New(path)
		if err != nil {
			return nil, err
		}
		client.configClient = c
	}

	// if there is an injected RepositoryFactory, use it, otherwise use a default one.
	if client.repositoryClientFactory == nil {
		client.repositoryClientFactory = defaultRepositoryFactory(client.configClient)
	}

	// if there is an injected ClusterFactory, use it, otherwise use a default one.
	if client.clusterClientFactory == nil {
		client.clusterClientFactory = defaultClusterFactory(client.configClient)
	}

	return client, nil
}

// defaultRepositoryFactory is a RepositoryClientFactory func the uses the default client provided by the repository low level library.
func defaultRepositoryFactory(configClient config.Client) RepositoryClientFactory {
	return func(input RepositoryClientFactoryInput) (repository.Client, error) {
		return repository.New(
			input.provider,
			configClient,
			repository.InjectYamlProcessor(input.processor),
		)
	}
}

// defaultClusterFactory is a ClusterClientFactory func the uses the default client provided by the cluster low level library.
func defaultClusterFactory(configClient config.Client) ClusterClientFactory {
	return func(input ClusterClientFactoryInput) (cluster.Client, error) {
		return cluster.New(
			// Kubeconfig is a type alias to cluster.Kubeconfig
			cluster.Kubeconfig(input.kubeconfig),
			configClient,
			cluster.InjectYamlProcessor(input.processor),
		), nil
	}
}
