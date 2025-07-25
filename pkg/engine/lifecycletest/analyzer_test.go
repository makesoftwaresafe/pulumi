// Copyright 2022-2024, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lifecycletest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/blang/semver"
	. "github.com/pulumi/pulumi/pkg/v3/engine" //nolint:revive
	lt "github.com/pulumi/pulumi/pkg/v3/engine/lifecycletest/framework"
	"github.com/pulumi/pulumi/pkg/v3/resource/deploy"
	"github.com/pulumi/pulumi/pkg/v3/resource/deploy/deploytest"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/config"
	"github.com/pulumi/pulumi/sdk/v3/go/common/resource/plugin"
	"github.com/pulumi/pulumi/sdk/v3/go/common/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testRequiredPolicy struct {
	name    string
	version string
	config  map[string]*json.RawMessage
}

func (p *testRequiredPolicy) Name() string {
	return p.name
}

func (p *testRequiredPolicy) Version() string {
	return p.version
}

func (p *testRequiredPolicy) Install(_ *plugin.Context) (string, error) {
	return "", nil
}

func (p *testRequiredPolicy) Config() map[string]*json.RawMessage {
	return p.config
}

func NewRequiredPolicy(name, version string, config map[string]*json.RawMessage) RequiredPolicy {
	return &testRequiredPolicy{
		name:    name,
		version: version,
		config:  config,
	}
}

func TestSimpleAnalyzer(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(opts *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			assert.Equal(t, "", opts.Organization)
			assert.Equal(t, "test-proj", opts.Project)
			assert.Equal(t, "test", opts.Stack)

			assert.Equal(t, map[config.Key]string{
				config.MustMakeKey(opts.Project, "bool"):   "true",
				config.MustMakeKey(opts.Project, "float"):  "1.5",
				config.MustMakeKey(opts.Project, "string"): "hello",
				config.MustMakeKey(opts.Project, "obj"):    "{\"key\":\"value\"}",
			}, opts.Config)

			return &deploytest.Analyzer{}, nil
		}),
	}

	programF := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		require.NoError(t, err)
		return nil
	})
	hostF := deploytest.NewPluginHostF(nil, nil, programF, loaders...)

	proj := "test-proj"
	p := &lt.TestPlan{
		Project: proj,
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: hostF,
		},
		Config: config.Map{
			config.MustMakeKey(proj, "bool"):   config.NewTypedValue("true", config.TypeBool),
			config.MustMakeKey(proj, "float"):  config.NewTypedValue("1.5", config.TypeFloat),
			config.MustMakeKey(proj, "string"): config.NewTypedValue("hello", config.TypeString),
			config.MustMakeKey(proj, "obj"):    config.NewObjectValue("{\"key\": \"value\"}"),
		},
	}

	project := p.GetProject()
	_, err := lt.TestOp(Update).Run(project, p.GetTarget(t, nil), p.Options, false, p.BackendClient, nil)
	require.NoError(t, err)
}

func TestSimpleAnalyzeResourceFailure(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				AnalyzeF: func(r plugin.AnalyzerResource) ([]plugin.AnalyzeDiagnostic, error) {
					return []plugin.AnalyzeDiagnostic{{
						PolicyName:       "always-fails",
						PolicyPackName:   "analyzerA",
						Description:      "a policy that always fails",
						Message:          "a policy failed",
						EnforcementLevel: apitype.Mandatory,
						URN:              r.URN,
					}}, nil
				},
			}, nil
		}),
	}

	programF := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		assert.Error(t, err)
		return nil
	})
	hostF := deploytest.NewPluginHostF(nil, nil, programF, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: hostF,
		},
	}

	project := p.GetProject()
	_, err := lt.TestOp(Update).Run(project, p.GetTarget(t, nil), p.Options, false, p.BackendClient, nil)
	assert.Error(t, err)
}

func TestSimpleAnalyzeStackFailure(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				AnalyzeStackF: func(rs []plugin.AnalyzerStackResource) ([]plugin.AnalyzeDiagnostic, error) {
					return []plugin.AnalyzeDiagnostic{{
						PolicyName:       "always-fails",
						PolicyPackName:   "analyzerA",
						Description:      "a policy that always fails",
						Message:          "a policy failed",
						EnforcementLevel: apitype.Mandatory,
						URN:              rs[0].URN,
					}}, nil
				},
			}, nil
		}),
	}

	programF := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		require.NoError(t, err)
		return nil
	})
	hostF := deploytest.NewPluginHostF(nil, nil, programF, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T:                t,
			SkipDisplayTests: true, // TODO: this seems flaky, could use some more investigation.
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: hostF,
		},
	}

	project := p.GetProject()
	_, err := lt.TestOp(Update).Run(project, p.GetTarget(t, nil), p.Options, false, p.BackendClient, nil)
	assert.Error(t, err)
}

// TestResourceRemediation tests a very simple sequence of remediations. We register two, to ensure that
// the remediations are applied in the order specified, an important part of the design.
func TestResourceRemediation(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				RemediateF: func(r plugin.AnalyzerResource) ([]plugin.Remediation, error) {
					// Run two remediations to ensure they are applied in order.
					return []plugin.Remediation{
						{
							PolicyName:        "ignored",
							PolicyPackName:    "analyzerA",
							PolicyPackVersion: "1.0.0",
							Description:       "a remediation that gets ignored because it runs first",
							Properties: resource.PropertyMap{
								"a":   resource.NewStringProperty("nope"),
								"ggg": resource.NewBoolProperty(true),
							},
						},
						{
							PolicyName:        "real-deal",
							PolicyPackName:    "analyzerA",
							PolicyPackVersion: "1.0.0",
							Description:       "a remediation that actually gets applied because it runs last",
							Properties: resource.PropertyMap{
								"a":   resource.NewStringProperty("foo"),
								"fff": resource.NewBoolProperty(true),
								"z":   resource.NewStringProperty("bar"),
							},
						},
					}, nil
				},
			}, nil
		}),
	}

	program := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		require.NoError(t, err)
		return nil
	})
	host := deploytest.NewPluginHostF(nil, nil, program, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: host,
		},
	}

	project := p.GetProject()
	snap, err := lt.TestOp(Update).Run(project, p.GetTarget(t, nil), p.Options, false, p.BackendClient, nil)

	// Expect no error, valid snapshot, two resources:
	assert.Nil(t, err)
	require.NotNil(t, snap)
	assert.Equal(t, 2, len(snap.Resources)) // stack plus pkA:m:typA

	// Ensure the rewritten properties have been applied to the inputs:
	r := snap.Resources[1]
	assert.Equal(t, "pkgA:m:typA", string(r.Type))
	assert.Equal(t, 3, len(r.Inputs))
	assert.Equal(t, "foo", r.Inputs["a"].StringValue())
	assert.Equal(t, true, r.Inputs["fff"].BoolValue())
	assert.Equal(t, "bar", r.Inputs["z"].StringValue())
}

// TestRemediationDiagnostic tests the case where a remediation issues a diagnostic rather than transforming
// state. In this case, the deployment should still succeed, even though no transforms took place.
func TestRemediationDiagnostic(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				RemediateF: func(r plugin.AnalyzerResource) ([]plugin.Remediation, error) {
					return []plugin.Remediation{{
						PolicyName:        "warning",
						PolicyPackName:    "analyzerA",
						PolicyPackVersion: "1.0.0",
						Description:       "a remediation with a diagnostic",
						Diagnostic:        "warning - could not run due to unknowns",
					}}, nil
				},
			}, nil
		}),
	}

	program := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		require.NoError(t, err)
		return nil
	})
	host := deploytest.NewPluginHostF(nil, nil, program, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: host,
		},
	}

	project := p.GetProject()
	snap, err := lt.TestOp(Update).Run(project, p.GetTarget(t, nil), p.Options, false, p.BackendClient, nil)

	// Expect no error, valid snapshot, two resources:
	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.Equal(t, 2, len(snap.Resources)) // stack plus pkA:m:typA
}

// TestRemediateFailure tests the case where a remediation fails to execute. In this case, the whole
// deployment itself should also fail.
func TestRemediateFailure(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				RemediateF: func(r plugin.AnalyzerResource) ([]plugin.Remediation, error) {
					return nil, errors.New("this remediation failed")
				},
			}, nil
		}),
	}

	program := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		assert.ErrorContains(t, err, "context canceled")
		return nil
	})
	host := deploytest.NewPluginHostF(nil, nil, program, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: host,
		},
	}

	project := p.GetProject()
	snap, res := lt.TestOp(Update).Run(project, p.GetTarget(t, nil), p.Options, false, p.BackendClient, nil)
	require.NotNil(t, res)
	require.NotNil(t, snap)
	assert.Equal(t, 0, len(snap.Resources))
}

func TestSimpleAnalyzeResourceFailureRemediateDowngradedToMandatory(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				AnalyzeF: func(r plugin.AnalyzerResource) ([]plugin.AnalyzeDiagnostic, error) {
					return []plugin.AnalyzeDiagnostic{{
						PolicyName:       "always-fails",
						PolicyPackName:   "analyzerA",
						Description:      "a policy that always fails",
						Message:          "a policy failed",
						EnforcementLevel: apitype.Remediate,
						URN:              r.URN,
					}}, nil
				},
			}, nil
		}),
	}

	programF := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		assert.Error(t, err)
		return nil
	})
	hostF := deploytest.NewPluginHostF(nil, nil, programF, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: hostF,
		},
		Steps: []lt.TestStep{
			{
				Op:            Update,
				SkipPreview:   true,
				ExpectFailure: true,
				Validate: func(project workspace.Project, target deploy.Target, entries JournalEntries,
					events []Event, err error,
				) error {
					violationEvents := []Event{}
					for _, e := range events {
						if e.Type == PolicyViolationEvent {
							violationEvents = append(violationEvents, e)
						}
					}
					require.Len(t, violationEvents, 1)
					assert.Equal(t, apitype.Mandatory,
						violationEvents[0].Payload().(PolicyViolationEventPayload).EnforcementLevel)

					return err
				},
			},
		},
	}

	p.Run(t, nil)
}

func TestSimpleAnalyzeStackFailureRemediateDowngradedToMandatory(t *testing.T) {
	t.Parallel()

	loaders := []*deploytest.PluginLoader{
		deploytest.NewProviderLoader("pkgA", semver.MustParse("1.0.0"), func() (plugin.Provider, error) {
			return &deploytest.Provider{}, nil
		}),
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				AnalyzeStackF: func(rs []plugin.AnalyzerStackResource) ([]plugin.AnalyzeDiagnostic, error) {
					return []plugin.AnalyzeDiagnostic{{
						PolicyName:       "always-fails",
						PolicyPackName:   "analyzerA",
						Description:      "a policy that always fails",
						Message:          "a policy failed",
						EnforcementLevel: apitype.Remediate,
						URN:              rs[0].URN,
					}}, nil
				},
			}, nil
		}),
	}

	programF := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		_, err := monitor.RegisterResource("pkgA:m:typA", "resA", true)
		require.NoError(t, err)
		return nil
	})
	hostF := deploytest.NewPluginHostF(nil, nil, programF, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T:                t,
			SkipDisplayTests: true, // TODO: this seems flaky, could use some more investigation.
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: hostF,
		},
		Steps: []lt.TestStep{
			{
				Op:            Update,
				SkipPreview:   true,
				ExpectFailure: true,
				Validate: func(project workspace.Project, target deploy.Target, entries JournalEntries,
					events []Event, err error,
				) error {
					violationEvents := []Event{}
					for _, e := range events {
						if e.Type == PolicyViolationEvent {
							violationEvents = append(violationEvents, e)
						}
					}
					require.Len(t, violationEvents, 1)
					assert.Equal(t, apitype.Mandatory,
						violationEvents[0].Payload().(PolicyViolationEventPayload).EnforcementLevel)

					return err
				},
			},
		},
	}

	p.Run(t, nil)
}

func TestAnalyzerCancellation(t *testing.T) {
	t.Parallel()

	gracefulShutdown := false
	loaders := []*deploytest.PluginLoader{
		deploytest.NewAnalyzerLoader("analyzerA", func(_ *plugin.PolicyAnalyzerOptions) (plugin.Analyzer, error) {
			return &deploytest.Analyzer{
				CancelF: func() error {
					gracefulShutdown = true
					return nil
				},
			}, nil
		}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	programF := deploytest.NewLanguageRuntimeF(func(_ plugin.RunInfo, monitor *deploytest.ResourceMonitor) error {
		time.Sleep(1 * time.Second)
		cancel()
		return nil
	})

	hostF := deploytest.NewPluginHostF(nil, nil, programF, loaders...)

	p := &lt.TestPlan{
		Options: lt.TestUpdateOptions{
			T: t,
			UpdateOptions: UpdateOptions{
				RequiredPolicies: []RequiredPolicy{NewRequiredPolicy("analyzerA", "", nil)},
			},
			HostF: hostF,
		},
	}
	project, target := p.GetProject(), p.GetTarget(t, nil)

	op := lt.TestOp(Update)
	_, err := op.RunWithContext(ctx, project, target, p.Options, false, nil, nil)

	assert.ErrorContains(t, err, "BAIL: canceled")
	assert.True(t, gracefulShutdown)
}
