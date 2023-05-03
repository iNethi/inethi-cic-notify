package task

import (
	"context"
	"encoding/json"

	"github.com/grassrootseconomics/cic-notify/internal/graphql"
	"github.com/grassrootseconomics/cic-notify/internal/notify"
	"github.com/grassrootseconomics/cic-notify/internal/templates"
	"github.com/hibiken/asynq"
)

type successReceivedMsg struct {
	ShortHash     string
	TransferValue uint64
	VoucherSymbol string
	ReceivedFrom  string
	DateString    string
	// These are passed to the channel provider e.g. AfricasTalking, Telegram, e.t.c.
	ChannelType       graphql.Interface_type_enum
	ChannelIdentifier string
}

func SuccessReceivedMsgProcessor(n *notify.Notify) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var (
			payload successReceivedMsg
		)

		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return err
		}

		templatePayload := struct {
			ShortHash      string
			TransferValue  uint64
			VoucherSymbol  string
			ReceivedFrom   string
			DateString     string
			CurrentBalance uint64
		}{
			payload.ShortHash,
			payload.TransferValue,
			payload.VoucherSymbol,
			payload.ReceivedFrom,
			payload.DateString,
			// TODO: Fetch current balance.
			0,
		}

		msgPayload, err := n.TxNotifyTemplates.Prepare(
			templates.SuccessReceivedTemplate,
			templatePayload,
		)
		if err != nil {
			return err
		}

		if err := routeMessage(
			ctx,
			n.TaskerClient,
			payload.ChannelType,
			msgPayload,
			payload.ChannelIdentifier,
		); err != nil {
			return nil
		}

		return nil
	}
}
