// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

//go:build celvalidation
// +build celvalidation

package celvalidation

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	egv1a1 "github.com/envoyproxy/gateway/api/v1alpha1"
	"github.com/envoyproxy/gateway/internal/utils/ptr"
)

func TestEnvoyProxyProvider(t *testing.T) {
	ctx := context.Background()
	baseEnvoyProxy := egv1a1.EnvoyProxy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "proxy",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: egv1a1.EnvoyProxySpec{},
	}

	cases := []struct {
		desc         string
		mutate       func(envoy *egv1a1.EnvoyProxy)
		mutateStatus func(envoy *egv1a1.EnvoyProxy)
		wantErrors   []string
	}{
		{
			desc: "nil provider",
			mutate: func(envoy *egv1a1.EnvoyProxy) {

			},
			wantErrors: []string{},
		},
		{
			desc: "unsupported provider",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: "foo",
					},
				}
			},
			wantErrors: []string{"Unsupported value: \"foo\": supported values: \"Kubernetes\""},
		},
		{
			desc: "invalid service type",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type: ptr.To(egv1a1.ServiceType("foo")),
							},
						},
					},
				}
			},
			wantErrors: []string{"Unsupported value: \"foo\": supported values: \"ClusterIP\", \"LoadBalancer\", \"NodePort\""},
		},
		{
			desc: "allocateLoadBalancerNodePorts-pass-case1",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type:                          ptr.To(egv1a1.ServiceTypeLoadBalancer),
								AllocateLoadBalancerNodePorts: ptr.To(true),
							},
						},
					},
				}
			},
			wantErrors: []string{},
		},
		{
			desc: "allocateLoadBalancerNodePorts-pass-case2",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type: ptr.To(egv1a1.ServiceTypeClusterIP),
							},
						},
					},
				}
			},
			wantErrors: []string{},
		},
		{
			desc: "allocateLoadBalancerNodePorts-fail",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type:                          ptr.To(egv1a1.ServiceTypeClusterIP),
								AllocateLoadBalancerNodePorts: ptr.To(true),
							},
						},
					},
				}
			},
			wantErrors: []string{"allocateLoadBalancerNodePorts can only be set for LoadBalancer type"},
		},
		{
			desc: "ServiceTypeLoadBalancer-with-valid-IP",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type:           ptr.To(egv1a1.ServiceTypeLoadBalancer),
								LoadBalancerIP: ptr.To("20.205.243.166"), // github ip for test only
							},
						},
					},
				}
			},
			wantErrors: []string{},
		},
		{
			desc: "ServiceTypeLoadBalancer-with-empty-IP",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type: ptr.To(egv1a1.ServiceTypeLoadBalancer),
							},
						},
					},
				}
			},
			wantErrors: []string{},
		},
		{
			desc: "ServiceTypeLoadBalancer-with-invalid-IP",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type:           ptr.To(egv1a1.ServiceTypeLoadBalancer),
								LoadBalancerIP: ptr.To("a.b.c.d"),
							},
						},
					},
				}
			},
			wantErrors: []string{"loadBalancerIP must be a valid IPv4 address"},
		},
		{
			desc: "ServiceTypeClusterIP-with-empty-IP",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type: ptr.To(egv1a1.ServiceTypeClusterIP),
							},
						},
					},
				}
			},
			wantErrors: []string{},
		},
		{
			desc: "ServiceTypeClusterIP-with-LoadBalancerIP",
			mutate: func(envoy *egv1a1.EnvoyProxy) {
				envoy.Spec = egv1a1.EnvoyProxySpec{
					Provider: &egv1a1.EnvoyProxyProvider{
						Type: egv1a1.ProviderTypeKubernetes,
						Kubernetes: &egv1a1.EnvoyProxyKubernetesProvider{
							EnvoyService: &egv1a1.KubernetesServiceSpec{
								Type:           ptr.To(egv1a1.ServiceTypeClusterIP),
								LoadBalancerIP: ptr.To("20.205.243.166"), // github ip for test only
							},
						},
					},
				}
			},
			wantErrors: []string{"loadBalancerIP can only be set for LoadBalancer type"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			proxy := baseEnvoyProxy.DeepCopy()
			proxy.Name = fmt.Sprintf("proxy-%v", time.Now().UnixNano())

			if tc.mutate != nil {
				tc.mutate(proxy)
			}
			err := c.Create(ctx, proxy)

			if tc.mutateStatus != nil {
				tc.mutateStatus(proxy)
				err = c.Status().Update(ctx, proxy)
			}

			if (len(tc.wantErrors) != 0) != (err != nil) {
				t.Fatalf("Unexpected response while creating EnvoyProxy; got err=\n%v\n;want error=%v", err, tc.wantErrors != nil)
			}

			var missingErrorStrings []string
			for _, wantError := range tc.wantErrors {
				if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(wantError)) {
					missingErrorStrings = append(missingErrorStrings, wantError)
				}
			}
			if len(missingErrorStrings) != 0 {
				t.Errorf("Unexpected response while creating EnvoyProxy; got err=\n%v\n;missing strings within error=%q", err, missingErrorStrings)
			}
		})
	}
}