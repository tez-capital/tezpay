package common

type ConfigurationVersionInfo struct {
	TPVersion *uint   `json:"tezpay_config_version,omitempty" yaml:"tezpay_config_version,omitempty"`
	Version   *string `json:"version,omitempty" yaml:"version,omitempty"`
}
