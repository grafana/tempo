// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package apmsampling

type SamplerConfig struct {
	AllEnvs SamplerEnvConfig `json:"all_envs"`
	ByEnv   []EnvAndConfig   `json:"by_env"`
}

type SamplerEnvConfig struct {
	PrioritySamplerTargetTPS *float64 `json:"priority_sampler_target_TPS"`
	ErrorsSamplerTargetTPS   *float64 `json:"errors_sampler_target_TPS"`
	RareSamplerEnabled       *bool    `json:"rare_sampler_enabled"`
}

type EnvAndConfig struct {
	Env    string           `json:"env"`
	Config SamplerEnvConfig `json:"config"`
}
