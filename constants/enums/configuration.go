package enums

type EConfigurationSeedKind string

const (
	TRD_CONFIGURATION_SEED EConfigurationSeedKind = "trd"
)

var (
	SUPPORTED_CONFIGURATION_SEED_KINDS = []EConfigurationSeedKind{
		TRD_CONFIGURATION_SEED,
	}
)
