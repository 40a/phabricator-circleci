package main

import (
	"github.com/aws/aws-sdk-go/service/sqs"
	"golang.org/x/net/context"
)

type parsedMessage interface {
	LooksValid() bool
	Execute(context.Context) error
	OriginalMsg() *sqs.Message
}

type msgConstructor func(*sqs.Message) (parsedMessage, error)

type msgProcessor struct {
	ch              <-chan *sqs.Message
	invalidMessages chan<- *sqs.Message
	parsedMsgs      chan<- parsedMessage

	closeSignal chan struct{}
	doneSignal  chan struct{}
	runErr      error
	parsers     []msgConstructor
}

func newMsgProcessor(ch <-chan *sqs.Message, invalidMessages chan<- *sqs.Message, parsedMsgs chan<- parsedMessage, parsers []msgConstructor) *msgProcessor {
	return &msgProcessor{
		ch:              ch,
		invalidMessages: invalidMessages,
		parsedMsgs:      parsedMsgs,
		parsers:         parsers,
	}
}

func (m *msgProcessor) Start(ctx context.Context) error {
	if m.closeSignal != nil {
		panic("Cannot start twice!")
	}

	m.closeSignal = make(chan struct{})
	m.doneSignal = make(chan struct{})

	go func() {
		m.runErr = runInCtx(ctx, m.processMessages)
		close(m.doneSignal)
	}()
	return nil
}

func (m *msgProcessor) Input() <-chan *sqs.Message {
	return m.ch
}

func (m *msgProcessor) Close() error {
	close(m.closeSignal)
	<-m.Done()
	return m.Err()
}

func (m *msgProcessor) Done() <-chan struct{} {
	return m.doneSignal
}

func (m *msgProcessor) Err() error {
	return m.runErr
}

func (m *msgProcessor) onMessage(ctx context.Context, msg *sqs.Message) error {
	for _, p := range m.parsers {
		if parsed, err := p(msg); err == nil {
			select {
			case m.parsedMsgs <- parsed:
			case <-ctx.Done():
				return ctx.Err()
			case <-m.closeSignal:
				return nil
			}
			return nil
		}
	}
	select {
	case m.invalidMessages <- msg:
	case <-ctx.Done():
		return ctx.Err()
	case <-m.closeSignal:
		return nil
	}
	return nil
}

func (m *msgProcessor) processMessages(ctx context.Context) error {
	l := getLog(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-m.ch:
			l.Printf("A message from chan: %s", *msg.MessageId)
			if err := m.onMessage(ctx, msg); err != nil {
				return err
			}
		}
	}
}
