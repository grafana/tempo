// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package sampler

// tracers with an env value of "" or agentEnv share
// the same sampler. This is required as remote is unaware
// of agentEnv and tracerEnv different values
func toSamplerEnv(tracerEnv, agentEnv string) string {
	env := tracerEnv
	if env == "" {
		env = agentEnv
	}
	return env
}

// tracers with empty env will have the same rate given
// as tracers with agentEnv
func rateWithEmptyEnv(samplerEnv, agentEnv string) bool {
	return samplerEnv == agentEnv
}
