package runner

import (
	"context"
	"log"

	tmCfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/rpc/client"
	coreTypes "github.com/tendermint/tendermint/rpc/core/types"
	"github.com/tendermint/tendermint/types"
)

type EventListener struct {
	client     client.Client
	TxChan     <-chan coreTypes.ResultEvent
	HeaderChan <-chan coreTypes.ResultEvent
	ctx        context.Context
	cancel     func()
}

func NewEventListener() *EventListener {
	tmClient := TMClient()
	ctx, cancel := context.WithCancel(context.Background())
	txChan := subscribeToEvent(tmClient, &ctx, types.EventTx)
	headerChan := subscribeToEvent(tmClient, &ctx, types.EventNewBlockHeader)
	return &EventListener{
		client:     tmClient,
		ctx:        ctx,
		TxChan:     txChan,
		HeaderChan: headerChan,
		cancel:     cancel,
	}
}

func subscribeToEvent(client client.Client, ctx *context.Context, evt string) <-chan coreTypes.ResultEvent {
	if !client.IsRunning() {
		_ = client.Start()
	}
	txChan, err := client.Subscribe(*ctx, "helpers", types.QueryForEvent(evt).String())
	if err != nil {
		log.Fatal(err)
	}
	return txChan
}

func TMClient() client.Client {
	client := client.NewHTTP(tmCfg.TestConfig().RPC.ListenAddress, "/websocket")
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
func (el *EventListener) Reset() *EventListener {
	el.Stop()
	return NewEventListener()
}
