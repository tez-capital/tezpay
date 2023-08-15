package enums

type EConfigurationSeedKind string

const (
	TRD_CONFIGURATION_SEED EConfigurationSeedKind = "trd"
	BC_CONFIGURATION_SEED  EConfigurationSeedKind = "bc"
)

var (
	SUPPORTED_CONFIGURATION_SEED_KINDS = []EConfigurationSeedKind{
		TRD_CONFIGURATION_SEED,
		BC_CONFIGURATION_SEED,
	}
)
