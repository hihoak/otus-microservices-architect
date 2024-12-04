package sagas

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/domain/order"
	"github.com/hihoak/otus-microservices-architect/cmd/order-service/sagas/create_order"
	kafka2 "github.com/hihoak/otus-microservices-architect/internal/adapters/kafka"
	"github.com/hihoak/otus-microservices-architect/internal/pkg/logger"
	"github.com/segmentio/kafka-go"
	"strings"
	"time"
)

type OrdersRepository interface {
	GetOrderByID(ctx context.Context, id order.OrderID) (*order.Order, error)
	UpdateOrder(ctx context.Context, ord order.Order) error
}

var (
	ErrFetchMessagesFailed = errors.New("fetch messages failed")
	ErrUnmarshalFailed     = errors.New("unmarshal failed")
)

type Orchestrator struct {
	ctx    context.Context
	cancel context.CancelFunc

	ordersRepository              OrdersRepository
	createOrderSagaEventsReader   *kafka.Reader
	createOrderSagaEventsProducer *kafka2.ClientCreateOrderSagaCommand
}

func NewOrchestrator(ctx context.Context, ordersRepository OrdersRepository, createOrderSagaEventsProducer *kafka2.ClientCreateOrderSagaCommand) *Orchestrator {
	ctx, cancel := context.WithCancel(ctx)
	return &Orchestrator{
		ctx:    ctx,
		cancel: cancel,

		ordersRepository: ordersRepository,

		createOrderSagaEventsProducer: createOrderSagaEventsProducer,
		createOrderSagaEventsReader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:               []string{"kafka.kafka.svc.cluster.local:9092"},
			GroupID:               "order-service",
			Topic:                 "create-order-saga-commands",
			MaxWait:               time.Millisecond * 200,
			JoinGroupBackoff:      time.Millisecond * 500,
			HeartbeatInterval:     time.Millisecond * 200,
			WatchPartitionChanges: true,
		}),
	}
}

func (o *Orchestrator) Start(ctx context.Context) {
	o.listenCreateOrderSagaEvents(ctx)
}

func (o *Orchestrator) Stop() {
	if err := o.createOrderSagaEventsReader.Close(); err != nil {
		logger.Log.Error("failed to close reader:", err)
	}
	if err := o.createOrderSagaEventsProducer.Close(); err != nil {
		logger.Log.Error("failed to close producer:", err)
	}
}

func (o *Orchestrator) fetchMessage() (*kafka2.CreateOrderSagaCommand, kafka.Message, error) {
	m, err := o.createOrderSagaEventsReader.FetchMessage(context.Background())
	if err != nil {
		return nil, kafka.Message{}, fmt.Errorf("fetch message: %w: %w", err, ErrFetchMessagesFailed)
	}
	var unmarshalledEvent kafka2.CreateOrderSagaCommand
	err = json.Unmarshal(m.Value, &unmarshalledEvent)
	if err != nil {
		return nil, kafka.Message{}, fmt.Errorf("unmarshall message: %w: %w", err, ErrUnmarshalFailed)
	}
	return &unmarshalledEvent, m, nil
}

func (o *Orchestrator) commitMessage(ctx context.Context, m kafka.Message) {
	if err := o.createOrderSagaEventsReader.CommitMessages(ctx, m); err != nil {
		logger.Log.Error("error committing messages: %v", err)
	}
}

func (o *Orchestrator) listenCreateOrderSagaEvents(ctx context.Context) {
	go func() {
		defer func() {
			o.Stop()
		}()
		for {
			select {
			case <-o.ctx.Done():
				return
			default:
				event, m, err := o.fetchMessage()
				if err != nil {
					if errors.Is(err, ErrUnmarshalFailed) {
						logger.Log.Warn("create order saga: fetch message message with wrong schema: %v", err)
						o.commitMessage(ctx, m)
						continue
					}
					logger.Log.Error("create order saga: fetch message: %v", err)
					continue
				}

				if strings.HasSuffix(string(event.Name), "Command") {
					o.commitMessage(ctx, m)
					continue
				}

				logger.Log.Info("consumed event: %v", event)
				ord, err := o.ordersRepository.GetOrderByID(ctx, order.OrderID(event.ID))
				if err != nil {
					if errors.Is(err, order.ErrNotFound) {
						logger.Log.Warn("create order saga: order not found")
						o.commitMessage(ctx, m)
						continue
					}
					logger.Log.Error("get order by id failed:", err)
					continue
				}

				logger.Log.Info("recreate order saga for status %v and move it to with event %v", ord.Status, event.Name)
				saga := create_order.InitCreateOrderSaga(ord.Status)
				if err := saga.Event(ctx, string(event.Name)); err != nil {
					if errors.Is(err, create_order.ErrNotFoundTransition) {
						logger.Log.Warn("create order saga: not found transition: %v", err)
						o.commitMessage(ctx, m)
						continue
					}
					logger.Log.Error("create order saga: unexpected err: %v", err)
					continue
				}

				nextCommand, err := saga.GetNextCommand()
				if err != nil {
					logger.Log.Error("for order %v: create order saga: get next command: %v", ord, err)
					continue
				}

				ord.Status = saga.Current()

				if err := o.ordersRepository.UpdateOrder(ctx, *ord); err != nil {
					logger.Log.Error("update order %v in repo: %v", ord, err)
					continue
				}

				if err := o.createOrderSagaEventsProducer.WriteEvent(ctx, nextCommand, ord); err != nil {
					logger.Log.Error("write event creat order saga %q: order %v", nextCommand, ord)
				}

				if err := o.createOrderSagaEventsReader.CommitMessages(ctx, m); err != nil {
					logger.Log.Error("error committing messages: %v", err)
				}
			}
		}
	}()
}
