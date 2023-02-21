// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package remoteconfighandler

import (
	"encoding/json"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state"
	"github.com/DataDog/datadog-agent/pkg/remoteconfig/state/products/apmsampling"
	"github.com/DataDog/datadog-agent/pkg/trace/config"
	"github.com/DataDog/datadog-agent/pkg/trace/log"
	"github.com/davecgh/go-spew/spew"
)

type prioritySampler interface {
	UpdateTargetTPS(targetTPS float64)
}

type errorsSampler interface {
	UpdateTargetTPS(targetTPS float64)
}

type rareSampler interface {
	SetEnabled(enabled bool)
}

// RemoteConfigHandler holds pointers to samplers that need to be updated when APM remote config changes
type RemoteConfigHandler struct {
	remoteClient    config.RemoteClient
	prioritySampler prioritySampler
	errorsSampler   errorsSampler
	rareSampler     rareSampler
	agentConfig     *config.AgentConfig
}

func New(conf *config.AgentConfig, prioritySampler prioritySampler, rareSampler rareSampler, errorsSampler errorsSampler) *RemoteConfigHandler {
	if conf.RemoteSamplingClient == nil {
		return nil
	}

	return &RemoteConfigHandler{
		remoteClient:    conf.RemoteSamplingClient,
		prioritySampler: prioritySampler,
		rareSampler:     rareSampler,
		errorsSampler:   errorsSampler,
		agentConfig:     conf,
	}
}

func (h *RemoteConfigHandler) Start() {
	if h == nil {
		return
	}

	h.remoteClient.Start()
	h.remoteClient.RegisterAPMUpdate(h.onUpdate)
}

func (h *RemoteConfigHandler) onUpdate(update map[string]state.APMSamplingConfig) {
	if len(update) == 0 {
		log.Debugf("no samplers configuration in remote config update payload")
		return
	}

	if len(update) > 1 {
		log.Errorf("samplers remote config payload contains %v configurations, but it should contain at most one", len(update))
		return
	}

	var samplerconfigPayload apmsampling.SamplerConfig
	for _, v := range update {
		err := json.Unmarshal(v.Config, &samplerconfigPayload)
		if err != nil {
			log.Error(err)
			return
		}
	}

	log.Debugf("updating samplers with remote configuration: %v", spew.Sdump(samplerconfigPayload))
	h.updateSamplers(samplerconfigPayload)
}

func (h *RemoteConfigHandler) updateSamplers(config apmsampling.SamplerConfig) {
	var confForEnv *apmsampling.SamplerEnvConfig
	for _, envAndConfig := range config.ByEnv {
		if envAndConfig.Env == h.agentConfig.DefaultEnv {
			confForEnv = &envAndConfig.Config
		}
	}

	var prioritySamplerTargetTPS float64
	if confForEnv != nil && confForEnv.PrioritySamplerTargetTPS != nil {
		prioritySamplerTargetTPS = *confForEnv.PrioritySamplerTargetTPS
	} else if config.AllEnvs.PrioritySamplerTargetTPS != nil {
		prioritySamplerTargetTPS = *config.AllEnvs.PrioritySamplerTargetTPS
	} else {
		prioritySamplerTargetTPS = h.agentConfig.TargetTPS
	}
	h.prioritySampler.UpdateTargetTPS(prioritySamplerTargetTPS)

	var errorsSamplerTargetTPS float64
	if confForEnv != nil && confForEnv.ErrorsSamplerTargetTPS != nil {
		errorsSamplerTargetTPS = *confForEnv.ErrorsSamplerTargetTPS
	} else if config.AllEnvs.ErrorsSamplerTargetTPS != nil {
		errorsSamplerTargetTPS = *config.AllEnvs.ErrorsSamplerTargetTPS
	} else {
		errorsSamplerTargetTPS = h.agentConfig.ErrorTPS
	}
	h.errorsSampler.UpdateTargetTPS(errorsSamplerTargetTPS)

	var rareSamplerEnabled bool
	if confForEnv != nil && confForEnv.RareSamplerEnabled != nil {
		rareSamplerEnabled = *confForEnv.RareSamplerEnabled
	} else if config.AllEnvs.RareSamplerEnabled != nil {
		rareSamplerEnabled = *config.AllEnvs.RareSamplerEnabled
	} else {
		rareSamplerEnabled = h.agentConfig.RareSamplerEnabled
	}
	h.rareSampler.SetEnabled(rareSamplerEnabled)
}
