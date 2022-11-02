package tzkt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"blockwatch.cc/tzgo/tezos"
	tezpay_tezos "github.com/alis-is/tezpay/clients/tezos"

	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
)

const (
	DELEGATOR_FETCH_LIMIT = 10000
)

type splitDelegator struct {
	Address string `json:"address"`
	Balance int64  `json:"balance"`
	Emptied bool   `json:"emptied,omitempty"`
}

type tzktBakersCycleData struct {
	StakingBalance     int64            `json:"stakingBalance"`
	DelegatedBalance   int64            `json:"delegatedBalance"`
	BlockRewards       int64            `json:"blockRewards"`
	EndorsementRewards int64            `json:"endorsementRewards"`
	NumDelegators      int32            `json:"numDelegators"`
	BlockFees          int64            `json:"blockFees"`
	Delegators         []splitDelegator `json:"delegators"`
}

type bakerData struct {
	FrozenDepositLimit int64 `json:"frozenDepositLimit"`
}

type Client struct {
	rootUrl *url.URL
}

func InitClient(rootUrl string) (*Client, error) {
	root, err := url.Parse(rootUrl)
	if err != nil {
		return nil, err
	}

	return &Client{
		rootUrl: root,
	}, nil
}

func (client *Client) Get(path string) (*http.Response, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(client.rootUrl.ResolveReference(rel).String())
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func unmarshallTzktResponse[T any](resp *http.Response, result *T) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read cycle data (tzkt) - %s", err.Error())
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return fmt.Errorf("failed to parse cycle data (tzkt) - %s", err.Error())
	}
	return nil
}

// https://api.tzkt.io/v1/rewards/split/${baker}/${cycle}?limit=${limit}&offset=${offset}
func (client *Client) getDelegatorsCycleData(baker []byte, cycle int64, limit int32, offset int) ([]splitDelegator, error) {
	u := fmt.Sprintf("rewards/split/%s/%d?limit=%d&offset=%d", baker, cycle, limit, offset)
	log.Debugf("getting delegators data of '%s' for cycle %d (%s)", baker, cycle, u)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cycle data (tzkt) - %s", err.Error())
	}
	data := &tzktBakersCycleData{}
	err = unmarshallTzktResponse(resp, data)
	if err != nil {
		return nil, err
	}
	return data.Delegators, nil
}

func (client *Client) getBakerData(baker []byte) (*bakerData, error) {
	u := fmt.Sprintf("delegates/%s", baker)
	log.Debugf("getting baker data of '%s' (%s)", baker, u)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cycle data (tzkt) - %s", err.Error())
	}
	data := &bakerData{}
	err = unmarshallTzktResponse(resp, data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (client *Client) getCycleData(baker []byte, cycle int64) (*tzktBakersCycleData, error) {
	u := fmt.Sprintf("rewards/split/%s/%d?limit=0", baker, cycle)
	log.Debugf("getting cycle data of '%s' for cycle %d (%s)", baker, cycle, u)
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cycle data (tzkt) - %s", err.Error())
	}

	tzktBakerCycleData := &tzktBakersCycleData{}
	err = unmarshallTzktResponse(resp, tzktBakerCycleData)
	if err != nil {
		return nil, err
	}
	return tzktBakerCycleData, nil
}

// https://api.tzkt.io/v1/rewards/split/${baker}/${cycle}?limit=0
func (client *Client) GetCycleData(baker tezos.Address, cycle int64) (bakersCycleData *tezpay_tezos.BakersCycleData, err error) {

	bakerAddr, _ := baker.MarshalText()

	tzktBakerData, err := client.getBakerData(bakerAddr)
	if err != nil {
		return nil, err
	}
	tzktBakerCycleData, err := client.getCycleData(bakerAddr, cycle)
	if err != nil {
		return nil, err
	}

	collectedDelegators := make([]splitDelegator, 0)
	fetched := DELEGATOR_FETCH_LIMIT
	for fetched == DELEGATOR_FETCH_LIMIT {
		newDelegators, err := client.getDelegatorsCycleData(bakerAddr, cycle, DELEGATOR_FETCH_LIMIT, len(collectedDelegators))
		if err != nil {
			return nil, err
		}
		collectedDelegators = append(collectedDelegators, newDelegators...)
		fetched = len(newDelegators)
	}

	// handle delegator parsing errors
	defer (func() {
		panicError := recover()
		if panicError != nil {
			err = panicError.(error)
			return
		}
	})()

	return &tezpay_tezos.BakersCycleData{
		StakingBalance:     tezos.NewZ(tzktBakerCycleData.StakingBalance),
		DelegatedBalance:   tezos.NewZ(tzktBakerCycleData.DelegatedBalance),
		BlockRewards:       tezos.NewZ(tzktBakerCycleData.BlockRewards),
		EndorsementRewards: tezos.NewZ(tzktBakerCycleData.EndorsementRewards),
		NumDelegators:      tzktBakerCycleData.NumDelegators,
		FrozenDeposit:      tezos.NewZ(tzktBakerData.FrozenDepositLimit),
		BlockFees:          tezos.NewZ(tzktBakerCycleData.BlockFees),
		Delegators: lo.Map(collectedDelegators, func(delegator splitDelegator, _ int) tezpay_tezos.Delegator {
			addr, err := tezos.ParseAddress(delegator.Address)
			if err != nil {
				panic(err)
			}
			return tezpay_tezos.Delegator{
				Address: addr,
				Balance: tezos.NewZ(delegator.Balance),
				Emptied: delegator.Emptied,
			}
		}),
	}, nil

}

// https://api.tzkt.io/v1/operations/transactions/onyUK7ZnQHzeNYbWSLL4zVATBtvLLk5GpPDv3VfoQPLtsBCjPX1/status
func (client *Client) WasOperationApplied(opHash tezos.OpHash) (bool, error) {
	op, _ := opHash.MarshalText()

	path := fmt.Sprintf("/operations/transactions/%s/status", op)
	resp, err := client.Get(path)
	if err != nil {
		return false, fmt.Errorf("failed to check operation stuats - %s", err.Error())
	}
	if resp.StatusCode == 204 {
		return false, nil
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to check operation status response body - %s", err.Error())
	}
	return bytes.Equal(body, []byte("true")), nil
}
