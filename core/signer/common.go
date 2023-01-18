package signer

import (
	"fmt"
	"os"
	"strings"

	"github.com/alis-is/tezpay/clients"
	"github.com/alis-is/tezpay/constants/enums"
	"github.com/alis-is/tezpay/core/common"
	"github.com/alis-is/tezpay/state"
	"github.com/hjson/hjson-go/v4"
	"github.com/sirupsen/logrus"
)

func Load(kind string) (common.SignerEngine, error) {
	switch kind {
	case string(enums.WALLET_MODE_LOCAL_PRIVATE_KEY2):
		fallthrough
	case string(enums.WALLET_MODE_LOCAL_PRIVATE_KEY):
		logrus.Debug("creating InMemorySigner")
		privateKeyFile := state.Global.GetPrivateKeyFilePath()
		logrus.Debugf("Loading private key from file '%s'", privateKeyFile)
		keyBytes, err := os.ReadFile(privateKeyFile)
		if err != nil {
			return nil, err
		}
		return clients.InitInMemorySigner(strings.TrimSpace(string(keyBytes)))
	case string(enums.WALLET_MODE_REMOTE_SIGNER2):
		fallthrough
	case string(enums.WALLET_MODE_REMOTE_SIGNER):
		logrus.Debug("creating RemoteSigner")
		remoteSpecsFile := state.Global.GetRemoteSpecsFilePath()
		logrus.Debugf("Loading remote specification from file '%s'", remoteSpecsFile)
		remoteSpecsBytes, err := os.ReadFile(remoteSpecsFile)
		if err != nil {
			return nil, err
		}
		remoteSpecs := clients.RemoteSignerSpecs{}
		err = hjson.Unmarshal(remoteSpecsBytes, &remoteSpecs)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal remote specs - %s", err.Error())
		}
		return clients.InitRemoteSignerFromSpecs(remoteSpecs)
	}

	if strings.HasPrefix(kind, "key:") {
		logrus.Debug("creating InMemorySigner from parameters")
		return clients.InitInMemorySigner(strings.TrimPrefix(kind, "key:"))
	}

	if strings.HasPrefix(kind, "remote:") {
		logrus.Debug("creating RemoteSigner from parameters")
		specs := strings.TrimPrefix(kind, "remote:")
		parts := strings.Split(specs, "@")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid remote specs '%s' from paramters", specs)
		}
		return clients.InitRemoteSigner(parts[0], parts[1])
	}

	return nil, fmt.Errorf("invalid payout wallet specification: '%s'", kind)
}
