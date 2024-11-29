package create_order

import (
	"context"
	"errors"
	"fmt"
	"github.com/looplab/fsm"
)

var (
	ErrNotFoundTransition  = errors.New("not found transition")
	ErrNotFoundNextCommand = errors.New("not found next command")
)

type CreateOrderSaga struct {
	stateMachine *fsm.FSM
}

type CreateOrderSagaEventType string

const (
	// states
	WithdrawMoneyPending     = "WithdrawMoneyPending"
	UndoWithdrawMoneyPending = "UndoWithdrawMoneyPending"
	ReserveStockPending      = "ReserveStockPending"
	UndoReserveStockPending  = "UndoReserveStockPending"
	ReserveSlotPending       = "ReserveSlotPending"
	NotifyPending            = "NotifyPending"
	OrderSucceeded           = "OrderSucceeded"
	OrderFailed              = "OrderFailed"

	// events
	WithdrawMoneySucceededEvent CreateOrderSagaEventType = "WithdrawMoneySucceededEvent"
	WithdrawMoneyFailedEvent    CreateOrderSagaEventType = "WithdrawMoneyFailedEvent"
	ReserveStockSucceededEvent  CreateOrderSagaEventType = "ReserveStockSucceededEvent"
	ReserveStockFailedEvent     CreateOrderSagaEventType = "ReserveStockFailedEvent"
	ReserveSlotSucceededEvent   CreateOrderSagaEventType = "ReserveSlotSucceededEvent"
	ReserveSlotFailedEvent      CreateOrderSagaEventType = "ReserveSlotFailedEvent"
	NotifySucceededEvent        CreateOrderSagaEventType = "NotifySucceededEvent"

	UndoWithdrawMoneySucceededEvent CreateOrderSagaEventType = "UndoWithdrawMoneySucceededEvent"
	UndoReserveStockSucceededEvent  CreateOrderSagaEventType = "UndoReserveStockSucceededEvent"

	// commands
	WithdrawMoneyCommand     = "WithdrawMoneyCommand"
	UndoWithdrawMoneyCommand = "UndoWithdrawMoneyCommand"
	ReserveStockCommand      = "ReserveStockCommand"
	UndoReserveStockCommand  = "UndoReserveStockCommand"
	ReserveSlotCommand       = "ReserveSlotCommand"
	NotifyCommand            = "NotifyCommand"
	NoCommand                = "NoCommand"
)

var stateToCommand = map[string]string{
	WithdrawMoneyPending:     WithdrawMoneyCommand,
	UndoWithdrawMoneyPending: UndoWithdrawMoneyCommand,
	ReserveStockPending:      ReserveStockCommand,
	UndoReserveStockPending:  UndoReserveStockCommand,
	ReserveSlotPending:       ReserveSlotCommand,
	NotifyPending:            NotifyCommand,
	OrderSucceeded:           NoCommand,
	OrderFailed:              NoCommand,
}

var createOrderStateMachineEvents = fsm.Events{
	// HAPPY PATH
	fsm.EventDesc{Name: string(WithdrawMoneySucceededEvent), Src: []string{WithdrawMoneyPending}, Dst: ReserveStockPending},
	fsm.EventDesc{Name: string(ReserveStockSucceededEvent), Src: []string{ReserveStockPending}, Dst: ReserveSlotPending},
	fsm.EventDesc{Name: string(ReserveSlotSucceededEvent), Src: []string{ReserveSlotPending}, Dst: NotifyPending},
	fsm.EventDesc{Name: string(NotifySucceededEvent), Src: []string{NotifyPending}, Dst: OrderSucceeded},

	// COMPENSATE PATH
	fsm.EventDesc{Name: string(WithdrawMoneyFailedEvent), Src: []string{WithdrawMoneyPending}, Dst: OrderFailed},
	fsm.EventDesc{Name: string(ReserveStockFailedEvent), Src: []string{ReserveStockPending}, Dst: UndoWithdrawMoneyPending},
	fsm.EventDesc{Name: string(UndoWithdrawMoneySucceededEvent), Src: []string{UndoWithdrawMoneyPending}, Dst: OrderFailed},
	fsm.EventDesc{Name: string(ReserveSlotFailedEvent), Src: []string{ReserveSlotPending}, Dst: UndoReserveStockPending},
	fsm.EventDesc{Name: string(UndoReserveStockSucceededEvent), Src: []string{UndoReserveStockPending}, Dst: UndoWithdrawMoneyPending},
}

func InitCreateOrderSaga(state string) *CreateOrderSaga {
	return &CreateOrderSaga{
		stateMachine: fsm.NewFSM(
			state,
			createOrderStateMachineEvents,
			map[string]fsm.Callback{},
		),
	}
}

func (c *CreateOrderSaga) Event(ctx context.Context, event string) error {
	if err := c.stateMachine.Event(ctx, event); err != nil {
		var invalidEventErr fsm.InvalidEventError
		if errors.Is(err, &invalidEventErr) {
			return fmt.Errorf("unknown event %s for state %s: %w", event, c.stateMachine.Current(), ErrNotFoundTransition)
		}
		return fmt.Errorf("proccess event: %w", err)
	}
	return nil
}

func (c *CreateOrderSaga) GetNextCommand() (string, error) {
	nextCommand, ok := stateToCommand[c.stateMachine.Current()]
	if !ok {
		return "", fmt.Errorf("state %s: %w", c.stateMachine.Current(), ErrNotFoundNextCommand)
	}
	return nextCommand, nil
}

func (c *CreateOrderSaga) Current() string {
	return c.stateMachine.Current()
}
