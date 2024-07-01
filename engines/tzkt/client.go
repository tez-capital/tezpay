package tzkt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/tez-capital/tezpay/common"
	"github.com/tez-capital/tezpay/constants"
	"github.com/tez-capital/tezpay/constants/enums"
	"github.com/tez-capital/tezpay/utils"
	"github.com/trilitech/tzgo/tezos"

	"github.com/samber/lo"
)

const (
	DELEGATOR_FETCH_LIMIT = 10000
)

type splitDelegator struct {
	Address string `json:"address"`

	DelegatedBalance int64 `json:"delegatedBalance"`
	StakedBalance    int64 `json:"stakedBalance"`

	Emptied bool `json:"emptied,omitempty"`
}

type tzktBakersCycleData struct {
	BakingPower              int64 `json:"bakingPower"`
	OwnDelegatedBalance      int64 `json:"ownDelegatedBalance"`
	ExternalDelegatedBalance int64 `json:"externalDelegatedBalance"`
	OwnStakedBalance         int64 `json:"ownStakedBalance"`      // OwnDelegatedBalance + ExternalDelegatedBalance
	ExternalStakedBalance    int64 `json:"externalStakedBalance"` // ExternalDelegatedBalance

	BlockRewardsDelegated  int64 `json:"blockRewardsDelegated"`
	BlockRewardsLiquid     int64 `json:"blockRewardsLiquid"`
	BlockRewardsStakedOwn  int64 `json:"blockRewardsStakedOwn"`
	BlockRewardsStakedEdge int64 `json:"blockRewardsStakedEdge"`
	// BlockRewards             int64            `json:"blockRewards"` // BlockRewardsLiquid + BlockRewardsStakedOwn
	MissedBlockRewards int64 `json:"missedBlockRewards"`

	EndorsementRewardsDelegated  int64 `json:"endorsementRewardsDelegated"`
	EndorsementRewardsLiquid     int64 `json:"endorsementRewardsLiquid"`
	EndorsementRewardsStakedOwn  int64 `json:"endorsementRewardsStakedOwn"`
	EndorsementRewardsStakedEdge int64 `json:"endorsementRewardsStakedEdge"`
	// EndorsementRewards       int64            `json:"endorsementRewards"` // EndorsementRewardsLiquid + EndorsementRewardsStakedOwn
	MissedEndorsementRewards int64 `json:"missedEndorsementRewards"`

	DelegatorsCount int32 `json:"delegatorsCount"`
	StakersCount    int32 `json:"stakersCount"`
	// NumDelegators            int32            `json:"numDelegators"` // DelegatorsCount
	BlockFees  int64            `json:"blockFees"`
	Delegators []splitDelegator `json:"delegators"`
}

type bakerData struct {
	FrozenDepositLimit int64 `json:"frozenDepositLimit"`
}

type Client struct {
	*http.Client

	rootUrl            *url.URL
	protocolRewardsUrl *url.URL
	balanceCheckMode   enums.EBalanceCheckMode
}

type TzktClientOptions struct {
	BalanceCheckMode enums.EBalanceCheckMode
	HttpClient       *http.Client
}

func InitClient(rootUrl string, protocolRewardsUrl string, options *TzktClientOptions) (*Client, error) {
	if options == nil {
		options = &TzktClientOptions{
			BalanceCheckMode: enums.PROTOCOL_BALANCE_CHECK_MODE,
		}
	}

	root, err := url.Parse(rootUrl)
	if err != nil {
		return nil, err
	}

	protocolRewards, err := url.Parse(protocolRewardsUrl)
	if err != nil {
		return nil, err
	}

	if options.HttpClient == nil {
		options.HttpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &Client{
		Client:             options.HttpClient,
		rootUrl:            root,
		protocolRewardsUrl: protocolRewards,
		balanceCheckMode:   options.BalanceCheckMode,
	}, nil
}

func (client *Client) Get(ctx context.Context, path string) (*http.Response, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	request, _ := http.NewRequestWithContext(ctx, "GET", client.rootUrl.ResolveReference(rel).String(), nil)

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (client *Client) GetFromProtocolRewards(ctx context.Context, path string) (*http.Response, error) {
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	request, _ := http.NewRequestWithContext(ctx, "GET", client.protocolRewardsUrl.ResolveReference(rel).String(), nil)

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func unmarshallTzktResponse[T any](resp *http.Response, result *T) error {
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return errors.Join(constants.ErrCycleDataUnmarshalFailed, err)
	}
	return nil
}

// https://api.tzkt.io/v1/rewards/split/${baker}/${cycle}?limit=${limit}&offset=${offset}
func (client *Client) getDelegatorsCycleData(ctx context.Context, baker []byte, cycle int64, limit int32, offset int) ([]splitDelegator, error) {
	u := fmt.Sprintf("v1/rewards/split/%s/%d?limit=%d&offset=%d", baker, cycle, limit, offset)
	slog.Debug("getting delegators data", "baker", baker, "cycle", cycle, "url", u)
	resp, err := client.Get(ctx, u)
	if err != nil {
		return nil, errors.Join(constants.ErrCycleDataFetchFailed, err)
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
	slog.Debug("getting baker data", "baker", baker, "url", u)
	resp, err := client.Get(ctx, u)
	if err != nil {
		return nil, errors.Join(constants.ErrCycleDataFetchFailed, err)
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
	slog.Debug("getting cycle data", "baker", baker, "cycle", cycle, "url", u)
	resp, err := client.Get(ctx, u)
	if err != nil {
		return nil, errors.Join(constants.ErrCycleDataFetchFailed, err)
	}
	if resp.StatusCode == 204 {
		return nil, errors.Join(constants.ErrNoCycleDataAvailable, fmt.Errorf("baker: %s", baker))
	}
	tzktBakerCycleData := &tzktBakersCycleData{}
	err = unmarshallTzktResponse(resp, tzktBakerCycleData)
	if err != nil {
		return nil, err
	}
	return tzktBakerCycleData, nil
}

func (client *Client) getProtocolRewardsCycleData(ctx context.Context, baker []byte, cycle int64) (*tzktBakersCycleData, error) {
	u := fmt.Sprintf("v1/rewards/split/%s/%d", baker, cycle)
	slog.Debug("getting protocol rewards cycle data", "baker", baker, "cycle", cycle, "url", u)
	resp, err := client.GetFromProtocolRewards(ctx, u)
	if err != nil {
		return nil, errors.Join(constants.ErrCycleDataProtocolRewardsFetchFailed, err)
	}
	if resp.StatusCode == 204 {
		return nil, errors.Join(constants.ErrNoCycleDataAvailable, fmt.Errorf("baker: %s", baker))
	}
	statusClass := resp.StatusCode / 100
	if statusClass != 2 {
		return nil, errors.Join(constants.ErrCycleDataProtocolRewardsFetchFailed, fmt.Errorf("status code: %d", resp.StatusCode))
	}
	tzktBakerCycleData := &tzktBakersCycleData{}
	err = unmarshallTzktResponse(resp, tzktBakerCycleData)
	if err != nil {
		return nil, err
	}
	return tzktBakerCycleData, nil
}

func (client *Client) getFirstBlockCycleAfterTimestamp(ctx context.Context, timestamp time.Time) (int64, error) {
	u := fmt.Sprintf("v1/blocks?select=cycle&limit=1&timestamp.gt=%s", timestamp.Format(time.RFC3339))
	slog.Debug("getting first block cycle after timestamp", "timestamp", timestamp, "url", u)
	resp, err := client.Get(ctx, u)
	if err != nil {
		return 0, errors.Join(constants.ErrCycleDataFetchFailed, err)
	}
	defer resp.Body.Close()
	var cycles []int64
	err = json.NewDecoder(resp.Body).Decode(&cycles)
	if err != nil {
		return 0, errors.Join(constants.ErrCycleDataUnmarshalFailed, err)
	}
	if len(cycles) == 0 {
		return 0, errors.Join(constants.ErrCycleDataFetchFailed, fmt.Errorf("no cycles found"))
	}
	return cycles[0], nil
}

// https://api.tzkt.io/v1/blocks?select=cycle,level&limit=1&timestamp.lt=2020-02-20T02:40:57Z
func (client *Client) GetCyclesInDateRange(ctx context.Context, startDate time.Time, endDate time.Time) ([]int64, error) {
	firstCycle, err := client.getFirstBlockCycleAfterTimestamp(ctx, startDate)
	if err != nil {
		return nil, err
	}
	firstCycleAfterTheRange, err := client.getFirstBlockCycleAfterTimestamp(ctx, endDate)
	if err != nil {
		return nil, err
	}

	cycles := make([]int64, 0, 20)
	for cycle := firstCycle; cycle < firstCycleAfterTheRange; cycle++ {
		cycles = append(cycles, cycle)
	}
	return cycles, nil
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
	slog.Debug("fetched baker data", "delegators_count", len(collectedDelegators))

	precision := int64(10000)

	var blockDelegatedRewards, endorsingDelegatedRewards, delegationShare tezos.Z
	firstAiActivatedCycle := constants.FIRST_PARIS_AI_ACTIVATED_CYCLE
	if cycle >= firstAiActivatedCycle || strings.Contains(client.rootUrl.Host, "ghostnet") {
		blockDelegatedRewards = tezos.NewZ(tzktBakerCycleData.BlockRewardsDelegated)
		endorsingDelegatedRewards = tezos.NewZ(tzktBakerCycleData.EndorsementRewardsDelegated)
		delegationShare = tezos.NewZ(tzktBakerCycleData.BakingPower - tzktBakerCycleData.OwnStakedBalance - tzktBakerCycleData.ExternalStakedBalance).Mul64(precision).Div64(tzktBakerCycleData.BakingPower)
	} else {
		blockDelegatedRewards = tezos.NewZ(tzktBakerCycleData.BlockRewardsLiquid).Add64(tzktBakerCycleData.BlockRewardsStakedOwn)
		endorsingDelegatedRewards = tezos.NewZ(tzktBakerCycleData.EndorsementRewardsLiquid).Add64(tzktBakerCycleData.EndorsementRewardsStakedOwn)
		delegationShare = tezos.NewZ(1)
		precision = 1
	}

	blockDelegatedFees := delegationShare.Mul64(tzktBakerCycleData.BlockFees).Div64(precision)
	blockStakingFees := tezos.NewZ(tzktBakerCycleData.BlockFees).Sub(blockDelegatedFees)

	if client.balanceCheckMode == enums.PROTOCOL_BALANCE_CHECK_MODE {
		protocolRewardsCycleData, err := client.getProtocolRewardsCycleData(ctx, bakerAddr, cycle)
		if err != nil {
			return nil, err
		}

		tzktBakerCycleData.OwnDelegatedBalance = protocolRewardsCycleData.OwnDelegatedBalance
		tzktBakerCycleData.ExternalDelegatedBalance = protocolRewardsCycleData.ExternalDelegatedBalance
		tzktBakerCycleData.OwnStakedBalance = protocolRewardsCycleData.OwnStakedBalance
		tzktBakerCycleData.ExternalStakedBalance = protocolRewardsCycleData.ExternalStakedBalance
		tzktBakerCycleData.DelegatorsCount = protocolRewardsCycleData.DelegatorsCount

		delegatorsMap := make(map[string]splitDelegator, len(protocolRewardsCycleData.Delegators))
		for _, delegator := range protocolRewardsCycleData.Delegators {
			delegatorsMap[delegator.Address] = delegator
		}

		// TODO: remove this when we confirm all works as expected
		var bakingPower tezos.Z
		delegatedPower := tezos.NewZ(tzktBakerCycleData.OwnDelegatedBalance).Add64(tzktBakerCycleData.ExternalDelegatedBalance)
		switch {
		case cycle > 750: // 751 is first cycle with baking power based on new staking model -> delegationPower is halved
			maximumDelegated := tezos.NewZ(tzktBakerCycleData.OwnStakedBalance).Mul64(9)
			if maximumDelegated.IsLess(delegatedPower) {
				delegatedPower = maximumDelegated
			}
			delegatedPower = delegatedPower.Div64(2)

			stakedPower := tezos.NewZ(tzktBakerCycleData.OwnStakedBalance).Add64(tzktBakerCycleData.ExternalStakedBalance)
			// we do not check maximum staking power, because overstake is automatically moved to delegation by protocol-rewards
			bakingPower = stakedPower.Add(delegatedPower)
		default:
			bakingPower = tezos.NewZ(tzktBakerCycleData.OwnStakedBalance).
				Add64(tzktBakerCycleData.ExternalStakedBalance).
				Add(delegatedPower)
			maximumBakingPower := tezos.NewZ(tzktBakerCycleData.OwnStakedBalance).Mul64(10)
			if maximumBakingPower.IsLess(bakingPower) {
				bakingPower = maximumBakingPower
			}
		}

		if utils.Abs(bakingPower.Int64()-tzktBakerCycleData.BakingPower) > 20 { // 20 mutes tolerance, we would be ok with 1 but since 751 when new model kicks in there is few more mutez difference, likely because of halved delegation power
			slog.Error("bakingPower mismatch", "bakingPower", bakingPower, "tzktBakerCycleData.BakingPower", tzktBakerCycleData.BakingPower)
			return nil, errors.Join(constants.ErrCycleDataProtocolRewardsMismatch, fmt.Errorf("bakingPower: %d, tzktBakerCycleData.BakingPower: %d, diff: %d", bakingPower.Int64(), tzktBakerCycleData.BakingPower, bakingPower.Int64()-tzktBakerCycleData.BakingPower))
		}
		// TODO: end remove this when we confirm all works as expected
		collectedDelegators = lo.Map(collectedDelegators, func(delegator splitDelegator, _ int) splitDelegator {
			if protocolRewardsDelegator, ok := delegatorsMap[delegator.Address]; ok {
				delegator.DelegatedBalance = protocolRewardsDelegator.DelegatedBalance
				delegator.StakedBalance = protocolRewardsDelegator.StakedBalance
				delete(delegatorsMap, delegator.Address) // remove from map to be able to check if there are any left
			} else {
				delegator.DelegatedBalance = 0
				delegator.StakedBalance = 0
			}
			return delegator
		})

		for _, delegator := range delegatorsMap {
			collectedDelegators = append(collectedDelegators, delegator)
		}
	}

	return &common.BakersCycleData{
		DelegatorsCount:                  tzktBakerCycleData.DelegatorsCount,
		OwnDelegatedBalance:              tezos.NewZ(tzktBakerCycleData.OwnDelegatedBalance),
		ExternalDelegatedBalance:         tezos.NewZ(tzktBakerCycleData.ExternalDelegatedBalance),
		BlockDelegatedRewards:            blockDelegatedRewards,
		IdealBlockDelegatedRewards:       blockDelegatedRewards.Add(delegationShare.Mul64(tzktBakerCycleData.MissedBlockRewards).Div64(precision)),
		EndorsementDelegatedRewards:      endorsingDelegatedRewards,
		IdealEndorsementDelegatedRewards: endorsingDelegatedRewards.Add(delegationShare.Mul64(tzktBakerCycleData.MissedEndorsementRewards).Div64(precision)),
		BlockDelegatedFees:               blockDelegatedFees,

		StakersCount:                  tzktBakerCycleData.StakersCount,
		OwnStakedBalance:              tezos.NewZ(tzktBakerCycleData.OwnStakedBalance),
		ExternalStakedBalance:         tezos.NewZ(tzktBakerCycleData.ExternalStakedBalance),
		BlockStakingRewardsEdge:       tezos.NewZ(tzktBakerCycleData.BlockRewardsStakedEdge),
		EndorsementStakingRewardsEdge: tezos.NewZ(tzktBakerCycleData.EndorsementRewardsStakedEdge),
		BlockStakingFees:              blockStakingFees,

		FrozenDepositLimit: tezos.NewZ(tzktBakerData.FrozenDepositLimit),
		Delegators: lo.Map(collectedDelegators, func(delegator splitDelegator, _ int) common.Delegator {
			addr, err := tezos.ParseAddress(delegator.Address)
			if err != nil {
				panic(err)
			}
			return common.Delegator{
				Address:          addr,
				DelegatedBalance: tezos.NewZ(delegator.DelegatedBalance),
				StakedBalance:    tezos.NewZ(delegator.StakedBalance),
				Emptied:          delegator.Emptied,
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
		return common.OPERATION_STATUS_UNKNOWN, errors.Join(constants.ErrOperationStatusCheckFailed, err)
	}
	if resp.StatusCode == 204 {
		return common.OPERATION_STATUS_NOT_EXISTS, nil
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return common.OPERATION_STATUS_UNKNOWN, errors.Join(constants.ErrOperationStatusCheckFailed, err)
	}
	if bytes.Equal(body, []byte("true")) {
		return common.OPERATION_STATUS_APPLIED, nil
	}
	if bytes.Equal(body, []byte("false")) {
		return common.OPERATION_STATUS_FAILED, nil
	}
	return common.OPERATION_STATUS_NOT_EXISTS, nil
}
