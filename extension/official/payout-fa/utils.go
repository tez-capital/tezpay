package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/tez-capital/tezpay/constants"
	"github.com/trilitech/tzgo/rpc"
)

func isClientSynced(ctx context.Context, client *rpc.Client) bool {
	status, err := client.GetStatus(ctx)
	return status.SyncState == "synced" || (err != nil && strings.Contains(err.Error(), "status 403"))
}

func InitializeRpcClients(ctx context.Context, rpc_urls []string, http_client *http.Client) ([]*rpc.Client, error) {
	rpc_clients := make([]*rpc.Client, 0, len(rpc_urls))
	chain_id := ""

	for _, rpcUrl := range rpc_urls {
		rpc_client, err := rpc.NewClient(rpcUrl, http_client)

		if err != nil {
			slog.Warn("failed to create rpc client", "rpc_url", rpcUrl, "error", err.Error())
			continue
		}
		rpc_clients = append(rpc_clients, rpc_client)

		client_chain_id := rpc_client.ChainId.String()
		if chain_id == "" {
			chain_id = client_chain_id
		}
		if chain_id != client_chain_id {
			return nil, constants.ErrMixedRpcs
		}
	}
	if len(rpc_clients) == 0 {
		return nil, fmt.Errorf("failed to create rpc clients, all %d failed", len(rpc_clients))
	} else if len(rpc_clients) < len(rpc_urls) {
		slog.Info(">>> at least one RPC client was successfully initialized - you can ignore above warnings <<<")
	}
	return rpc_clients, nil
}

func AttemptWithRpcClients[T any](ctx context.Context, clients []*rpc.Client, f func(client *rpc.Client) (T, error)) (T, error) {
	var err error
	var result T
	for _, client := range clients {
		if !isClientSynced(ctx, client) {
			continue
		}
		slog.Debug("attempting with client", "client", client.BaseURL.Host)

		result, err = f(client)
		if err != nil {
			continue
		}
		return result, nil
	}

	return result, errors.New("all clients failed")
}
