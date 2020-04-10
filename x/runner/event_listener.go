package runner

import (
	"context"
	"fmt"
	"log"

	// tmCfg "github.com/tendermint/tendermint/config"
	"github.com/pokt-network/pocket-runner/internal/types"
	"github.com/tendermint/tendermint/rpc/client"
	coreTypes "github.com/tendermint/tendermint/rpc/core/types"
	tmTypes"github.com/tendermint/tendermint/types"
)

const defaultListenAddr = "tcp://0.0.0.0:"
type EventListener struct {
	client     client.Client
	TxChan     <-chan coreTypes.ResultEvent
	HeaderChan <-chan coreTypes.ResultEvent
	ctx        context.Context
	cancel     func()
}

func NewEventListener(cfg *types.Config) *EventListener {
	tmClient := TMClient(cfg.GetPort())
	ctx, cancel := context.WithCancel(context.Background())
	txChan := subscribeToEvent(tmClient, ctx, tmTypes.EventTx)
	headerChan := subscribeToEvent(tmClient, ctx, tmTypes.EventNewBlockHeader)
	return &EventListener{
		client:     tmClient,
		ctx:        ctx,
		TxChan:     txChan,
		HeaderChan: headerChan,
		cancel:     cancel,
	}
}

func subscribeToEvent(client client.Client, ctx context.Context, evt string) <-chan coreTypes.ResultEvent {
	if !client.IsRunning() {
		_ = client.Start()
	}
	txChan, err := client.Subscribe(ctx, "helpers", tmTypes.QueryForEvent(evt).String())
	if err != nil {
		log.Fatal(err)
	}
	return txChan
}

func TMClient(port string) client.Client {
	address := fmt.Sprintf("%s%s", defaultListenAddr, port)
	fmt.Printf("opening client to %s\n", address)
	client := client.NewHTTP(address, "/websocket")
	return client
}

// Stop Listening
func (el *EventListener) Stop() {
	err := el.client.UnsubscribeAll(el.ctx, "helpers")
	if err != nil {
		log.Fatal(err)
	}
	err = el.client.Stop()
	if err != nil {
		log.Fatal(err)
	}
	el.cancel()
}
func (el *EventListener) Reset(cfg *types.Config) *EventListener {
	el.Stop()
	return NewEventListener(cfg)
}
