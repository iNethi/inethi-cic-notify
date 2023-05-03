package sub

import (
	"context"
	"errors"

	"github.com/grassrootseconomics/cic-custodial/pkg/util"
	"github.com/grassrootseconomics/cic-notify/internal/tasker"
	"github.com/nats-io/nats.go"
	"github.com/zerodha/logf"
)

const (
	durableId   = "cic-notify"
	pullStream  = "CHAIN"
	pullSubject = "CHAIN.transfer"
)

type (
	SubOpts struct {
		JsCtx        nats.JetStreamContext
		Logg         logf.Logger
		NatsConn     *nats.Conn
		TaskerClient *tasker.TaskerClient
	}

	Sub struct {
		jsCtx        nats.JetStreamContext
		logg         logf.Logger
		natsConn     *nats.Conn
		taskerClient *tasker.TaskerClient
	}
)

func NewSub(o SubOpts) (*Sub, error) {
	_, err := o.JsCtx.AddConsumer(pullStream, &nats.ConsumerConfig{
		Durable:       durableId,
		AckPolicy:     nats.AckExplicitPolicy,
		FilterSubject: pullSubject,
	})
	if err != nil {
		return nil, err
	}

	return &Sub{
		jsCtx:        o.JsCtx,
		logg:         o.Logg,
		natsConn:     o.NatsConn,
		taskerClient: o.TaskerClient,
	}, nil
}

func (s *Sub) Process() error {
	subOpts := []nats.SubOpt{
		nats.ManualAck(),
		nats.Bind(pullStream, durableId),
	}

	natsSub, err := s.jsCtx.PullSubscribe(pullSubject, durableId, subOpts...)
	if err != nil {
		return err
	}

	for {
		events, err := natsSub.Fetch(1)
		if err != nil {
			if errors.Is(err, nats.ErrTimeout) {
				continue
			} else if errors.Is(err, nats.ErrConnectionClosed) {
				return nil
			} else {
				return err
			}
		}

		if len(events) > 0 {
			msg := events[0]
			ctx, cancel := context.WithTimeout(context.Background(), util.SLATimeout)

			if err := s.processEventHandler(ctx, msg); err != nil {
				s.logg.Error("sub: handler error", "error", err)
				msg.Nak()
			} else {
				msg.Ack()
			}
			cancel()
		}
	}
}

func (s *Sub) Close() {
	if s.natsConn != nil {
		s.natsConn.Close()
	}
}
