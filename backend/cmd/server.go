/*
Copyright 2025 The Kubernetes Authors.

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

package main

import (
	"os"
	"strings"

	"github.com/kubernetes-sigs/headlamp/backend/pkg/cache"
	"github.com/kubernetes-sigs/headlamp/backend/pkg/config"
	headlampconfig "github.com/kubernetes-sigs/headlamp/backend/pkg/headlampconfig"
	"github.com/kubernetes-sigs/headlamp/backend/pkg/kubeconfig"
	"github.com/kubernetes-sigs/headlamp/backend/pkg/logger"
	"github.com/kubernetes-sigs/headlamp/backend/pkg/plugins"
)

func main() {
	if len(os.Args) == 2 && os.Args[1] == "list-plugins" {
		runListPlugins()

		return
	}

	conf, err := config.Parse(os.Args)
	if err != nil {
		logger.Log(logger.LevelError, nil, err, "fetching config:%v")
		os.Exit(1)
	}

	cache := cache.New[interface{}]()
	kubeConfigStore := kubeconfig.NewContextStore()
	multiplexer := NewMultiplexer(kubeConfigStore)

	StartHeadlampServer(&HeadlampConfig{
		HeadlampCFG: &headlampconfig.HeadlampCFG{
			UseInCluster:          conf.InCluster,
			KubeConfigPath:        conf.KubeConfigPath,
			SkippedKubeContexts:   conf.SkippedKubeContexts,
			ListenAddr:            conf.ListenAddr,
			Port:                  conf.Port,
			DevMode:               conf.DevMode,
			StaticDir:             conf.StaticDir,
			Insecure:              conf.InsecureSsl,
			PluginDir:             conf.PluginsDir,
			EnableHelm:            conf.EnableHelm,
			EnableDynamicClusters: conf.EnableDynamicClusters,
			WatchPluginsChanges:   conf.WatchPluginsChanges,
			KubeConfigStore:       kubeConfigStore,
			BaseURL:               conf.BaseURL,
			ProxyURLs:             strings.Split(conf.ProxyURLs, ","),
			TelemetryConfig: config.Config{
				ServiceName:        conf.ServiceName,
				ServiceVersion:     conf.ServiceVersion,
				TracingEnabled:     conf.TracingEnabled,
				MetricsEnabled:     conf.MetricsEnabled,
				JaegerEndpoint:     conf.JaegerEndpoint,
				OTLPEndpoint:       conf.OTLPEndpoint,
				UseOTLPHTTP:        conf.UseOTLPHTTP,
				StdoutTraceEnabled: conf.StdoutTraceEnabled,
				SamplingRate:       conf.SamplingRate,
			},
			OidcClientID:              conf.OidcClientID,
			OidcValidatorClientID:     conf.OidcValidatorClientID,
			OidcClientSecret:          conf.OidcClientSecret,
			OidcIdpIssuerURL:          conf.OidcIdpIssuerURL,
			OidcValidatorIdpIssuerURL: conf.OidcValidatorIdpIssuerURL,
			OidcScopes:                strings.Split(conf.OidcScopes, ","),
			OidcUseAccessToken:        conf.OidcUseAccessToken,
			Cache:                     cache,
		},
		multiplexer: multiplexer,
	})
}

func runListPlugins() {
	conf, err := config.Parse(os.Args[2:])
	if err != nil {
		logger.Log(logger.LevelError, nil, err, "fetching config:%v")
		os.Exit(1)
	}

	if err := plugins.ListPlugins(conf.StaticDir, conf.PluginsDir); err != nil {
		logger.Log(logger.LevelError, nil, err, "listing plugins")
	}
}
