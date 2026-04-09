// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package config

import (
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	KeyElsevierAPIKey    = "elsevier_api_key"
	KeyOpenAIAuditDir    = "openai_audit_dir"
	KeyOpenAIAuditEnable = "openai_audit_enabled"
	KeyOpenRouterAPIKey  = "openrouter_api_key"
	KeyOpenRouterBaseURL = "openrouter_base_url"
	KeyShirtyAPIKey      = "shirty_api_key"
	KeyShirtyBaseURL     = "shirty_base_url"

	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	DefaultShirtyBaseURL     = "https://shirty.sandia.gov/api/v1"
)

type Settings struct {
	ElsevierAPIKey    string
	OpenAIAuditDir    string
	OpenAIAuditEnable bool
	OpenRouterAPIKey  string
	OpenRouterBaseURL string
	ShirtyAPIKey      string
	ShirtyBaseURL     string
}

var runtimeConfig = viper.New()

func init() {
	runtimeConfig.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtimeConfig.AutomaticEnv()
	runtimeConfig.SetDefault(KeyOpenAIAuditEnable, true)
	runtimeConfig.SetDefault(KeyOpenRouterBaseURL, DefaultOpenRouterBaseURL)
	runtimeConfig.SetDefault(KeyShirtyBaseURL, DefaultShirtyBaseURL)
}

func BindFlags(flags *pflag.FlagSet) error {
	for key, flagName := range map[string]string{
		KeyElsevierAPIKey:    "elsevier-api-key",
		KeyOpenAIAuditDir:    "openai-audit-dir",
		KeyOpenAIAuditEnable: "openai-audit-enabled",
		KeyOpenRouterAPIKey:  "openrouter-api-key",
		KeyOpenRouterBaseURL: "openrouter-base-url",
		KeyShirtyAPIKey:      "shirty-api-key",
		KeyShirtyBaseURL:     "shirty-base-url",
	} {
		if err := runtimeConfig.BindPFlag(key, flags.Lookup(flagName)); err != nil {
			return err
		}
	}

	for key, envName := range map[string]string{
		KeyElsevierAPIKey:    "ELSEVIER_API_KEY",
		KeyOpenAIAuditDir:    "OPENAI_AUDIT_DIR",
		KeyOpenAIAuditEnable: "OPENAI_AUDIT_ENABLED",
		KeyOpenRouterAPIKey:  "OPENROUTER_API_KEY",
		KeyOpenRouterBaseURL: "OPENROUTER_BASE_URL",
		KeyShirtyAPIKey:      "SHIRTY_API_KEY",
		KeyShirtyBaseURL:     "SHIRTY_BASE_URL",
	} {
		if err := runtimeConfig.BindEnv(key, envName); err != nil {
			return err
		}
	}

	return nil
}

func Runtime() Settings {
	return Settings{
		ElsevierAPIKey:    runtimeConfig.GetString(KeyElsevierAPIKey),
		OpenAIAuditDir:    runtimeConfig.GetString(KeyOpenAIAuditDir),
		OpenAIAuditEnable: runtimeConfig.GetBool(KeyOpenAIAuditEnable),
		OpenRouterAPIKey:  runtimeConfig.GetString(KeyOpenRouterAPIKey),
		OpenRouterBaseURL: runtimeConfig.GetString(KeyOpenRouterBaseURL),
		ShirtyAPIKey:      runtimeConfig.GetString(KeyShirtyAPIKey),
		ShirtyBaseURL:     runtimeConfig.GetString(KeyShirtyBaseURL),
	}
}
