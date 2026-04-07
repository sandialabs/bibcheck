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
	KeyOpenRouterAPIKey  = "openrouter_api_key"
	KeyOpenRouterBaseURL = "openrouter_base_url"
	KeyShirtyAPIKey      = "shirty_api_key"
	KeyShirtyBaseURL     = "shirty_base_url"

	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	DefaultShirtyBaseURL     = "https://shirty.sandia.gov/api/v1"
)

type Settings struct {
	ElsevierAPIKey    string
	OpenRouterAPIKey  string
	OpenRouterBaseURL string
	ShirtyAPIKey      string
	ShirtyBaseURL     string
}

var runtimeConfig = viper.New()

func init() {
	runtimeConfig.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	runtimeConfig.AutomaticEnv()
	runtimeConfig.SetDefault(KeyOpenRouterBaseURL, DefaultOpenRouterBaseURL)
	runtimeConfig.SetDefault(KeyShirtyBaseURL, DefaultShirtyBaseURL)
}

func BindFlags(flags *pflag.FlagSet) error {
	for key, flagName := range map[string]string{
		KeyElsevierAPIKey:    "elsevier-api-key",
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
		OpenRouterAPIKey:  runtimeConfig.GetString(KeyOpenRouterAPIKey),
		OpenRouterBaseURL: runtimeConfig.GetString(KeyOpenRouterBaseURL),
		ShirtyAPIKey:      runtimeConfig.GetString(KeyShirtyAPIKey),
		ShirtyBaseURL:     runtimeConfig.GetString(KeyShirtyBaseURL),
	}
}
