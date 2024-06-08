package signer_engines

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hjson/hjson-go/v4"
	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/state"
)

func Load(kind string) (common.SignerEngine, error) {
	switch kind {
	case string(enums.WALLET_MODE_LOCAL_PRIVATE_KEY2):
		fallthrough
	case string(enums.WALLET_MODE_LOCAL_PRIVATE_KEY):
		slog.Debug("creating InMemorySigner")
		privateKeyFile := state.Global.GetPrivateKeyFilePath()
		slog.Debug("loading private key from file", "path", privateKeyFile)
		keyBytes, err := os.ReadFile(privateKeyFile)
		if err != nil {
			return nil, errors.Join(constants.ErrSignerLoadFailed, err)
		}
		return InitInMemorySigner(strings.TrimSpace(string(keyBytes)))
	case string(enums.WALLET_MODE_REMOTE_SIGNER2):
		fallthrough
	case string(enums.WALLET_MODE_REMOTE_SIGNER):
		slog.Debug("creating RemoteSigner")
		remoteSpecsFile := state.Global.GetRemoteSpecsFilePath()
		slog.Debug("loading remote specification from file", "path", remoteSpecsFile)
		remoteSpecsBytes, err := os.ReadFile(remoteSpecsFile)
		if err != nil {
			return nil, errors.Join(constants.ErrSignerLoadFailed, err)
		}
		remoteSpecs := RemoteSignerSpecs{}
		err = hjson.Unmarshal(remoteSpecsBytes, &remoteSpecs)
		if err != nil {
			return nil, errors.Join(constants.ErrSignerLoadFailed, errors.New("failed to unmarshal remote specs"), err)
		}
		return InitRemoteSignerFromSpecs(remoteSpecs)
	}

	if strings.HasPrefix(kind, "key:") {
		slog.Debug("creating InMemorySigner from parameters")
		return InitInMemorySigner(strings.TrimPrefix(kind, "key:"))
	}

	if strings.HasPrefix(kind, "remote:") {
		slog.Debug("creating RemoteSigner from parameters")
		specs := strings.TrimPrefix(kind, "remote:")
		parts := strings.Split(specs, "@")
		if len(parts) != 2 {
			return nil, errors.Join(constants.ErrSignerLoadFailed, fmt.Errorf("invalid remote specs '%s'", specs))
		}
		return InitRemoteSigner(parts[0], parts[1])
	}

	return nil, errors.Join(constants.ErrSignerLoadFailed, fmt.Errorf("invalid payout wallet specification: '%s'", kind))
}
