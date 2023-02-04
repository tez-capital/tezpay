package tzkt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"blockwatch.cc/tzgo/tezos"
	"github.com/alis-is/tezpay/core/common"

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
	StakingBalance           int64            `json:"stakingBalance"`
	DelegatedBalance         int64            `json:"delegatedBalance"`
	BlockRewards             int64            `json:"blockRewards"`
	MissedBlockRewards       int64            `json:"missedBlockRewards"`
	EndorsementRewards       int64            `json:"endorsementRewards"`
	MissedEndorsementRewards int64            `json:"missedEndorsementRewards"`
	NumDelegators            int32            `json:"numDelegators"`
	BlockFees                int64            `json:"blockFees"`
	Delegators               []splitDelegator `json:"delegators"`
}

type bakerData struct {
	FrozenDepositLimit int64 `json:"frozenDepositLimit"`
}

type Client struct {
	rootUrl    *url.URL
	httpClient *http.Client
}

func InitClient(rootUrl string, httpClient *http.Client) (*Client, error) {
	root, err := url.Parse(rootUrl)
	if err != nil {
		return nil, err
	}

	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		rootUrl:    root,
		httpClient: httpClient,
	}, nil
}

func (client *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	request, _ := http.NewRequestWithContext(ctx, "GET", client.rootUrl.ResolveReference(rel).String(), nil)

	resp, err := client.httpClient.Do(request)
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
func (client *Client) getDelegatorsCycleData(ctx context.Context, baker []byte, cycle int64, limit int32, offset int) ([]splitDelegator, error) {
	u := fmt.Sprintf("v1/rewards/split/%s/%d?limit=%d&offset=%d", baker, cycle, limit, offset)
	log.Debugf("getting delegators data of '%s' for cycle %d (%s)", baker, cycle, u)
	resp, err := client.Get(ctx, u)
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

func (client *Client) getBakerData(ctx context.Context, baker []byte) (*bakerData, error) {
	u := fmt.Sprintf("v1/delegates/%s", baker)
	log.Debugf("getting baker data of '%s' (%s)", baker, u)
	resp, err := client.Get(ctx, u)
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

func (client *Client) getCycleData(ctx context.Context, baker []byte, cycle int64) (*tzktBakersCycleData, error) {
	u := fmt.Sprintf("v1/rewards/split/%s/%d?limit=0", baker, cycle)
	log.Debugf("getting cycle data of '%s' for cycle %d (%s)", baker, cycle, u)
	resp, err := client.Get(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch cycle data (tzkt) - %s", err.Error())
	}
	if resp.StatusCode == 204 {
		return nil, fmt.Errorf("no cycle data available for baker '%s'", baker)
	}
	tzktBakerCycleData := &tzktBakersCycleData{}
	err = unmarshallTzktResponse(resp, tzktBakerCycleData)
	if err != nil {
		return nil, err
	}
	return tzktBakerCycleData, nil
}

// https://api.tzkt.io/v1/rewards/split/${baker}/${cycle}?limit=0
func (client *Client) GetCycleData(ctx context.Context, baker tezos.Address, cycle int64) (bakersCycleData *common.BakersCycleData, err error) {

	bakerAddr, _ := baker.MarshalText()

	tzktBakerData, err := client.getBakerData(ctx, bakerAddr)
	if err != nil {
		return nil, err
	}
	tzktBakerCycleData, err := client.getCycleData(ctx, bakerAddr, cycle)
	if err != nil {
		return nil, err
	}

	collectedDelegators := make([]splitDelegator, 0)
	fetched := DELEGATOR_FETCH_LIMIT
	for fetched == DELEGATOR_FETCH_LIMIT {
		newDelegators, err := client.getDelegatorsCycleData(ctx, bakerAddr, cycle, DELEGATOR_FETCH_LIMIT, len(collectedDelegators))
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
	log.Tracef("fetched baker data with %d delegators", len(collectedDelegators))

	return &common.BakersCycleData{
		StakingBalance:          tezos.NewZ(tzktBakerCycleData.StakingBalance),
		DelegatedBalance:        tezos.NewZ(tzktBakerCycleData.DelegatedBalance),
		BlockRewards:            tezos.NewZ(tzktBakerCycleData.BlockRewards),
		IdealBlockRewards:       tezos.NewZ(tzktBakerCycleData.BlockRewards).Add64(tzktBakerCycleData.MissedBlockRewards),
		EndorsementRewards:      tezos.NewZ(tzktBakerCycleData.EndorsementRewards),
		IdealEndorsementRewards: tezos.NewZ(tzktBakerCycleData.EndorsementRewards).Add64(tzktBakerCycleData.MissedEndorsementRewards),
		NumDelegators:           tzktBakerCycleData.NumDelegators,
		FrozenDepositLimit:      tezos.NewZ(tzktBakerData.FrozenDepositLimit),
		BlockFees:               tezos.NewZ(tzktBakerCycleData.BlockFees),
		Delegators: lo.Map(collectedDelegators, func(delegator splitDelegator, _ int) common.Delegator {
			addr, err := tezos.ParseAddress(delegator.Address)
			if err != nil {
				panic(err)
			}
			return common.Delegator{
				Address: addr,
				Balance: tezos.NewZ(delegator.Balance),
				Emptied: delegator.Emptied,
			}
		}),
	}, nil

}

// https://api.tzkt.io/v1/operations/transactions/onyUK7ZnQHzeNYbWSLL4zVATBtvLLk5GpPDv3VfoQPLtsBCjPX1/status
func (client *Client) WasOperationApplied(ctx context.Context, opHash tezos.OpHash) (common.OperationStatus, error) {
	op, _ := opHash.MarshalText()

	path := fmt.Sprintf("v1/operations/transactions/%s/status", op)
	resp, err := client.Get(ctx, path)
	if err != nil {
		return common.OPERATION_STATUS_UNKNOWN, fmt.Errorf("failed to check operation stuats - %s", err.Error())
	}
	if resp.StatusCode == 204 {
		return common.OPERATION_STATUS_NOT_EXISTS, nil
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return common.OPERATION_STATUS_UNKNOWN, fmt.Errorf("failed to check operation status response body - %s", err.Error())
	}
	if bytes.Equal(body, []byte("true")) {
		return common.OPERATION_STATUS_APPLIED, nil
	}
	if bytes.Equal(body, []byte("false")) {
		return common.OPERATION_STATUS_FAILED, nil
	}
	return common.OPERATION_STATUS_NOT_EXISTS, nil
}
