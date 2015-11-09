package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"golang.org/x/net/context"
	"sync"
)

type queuePoller struct {
	cfg               *aws.Config
	queueURL          string
	waitTimeSeconds   int64
	msgRemoveLog      logger
	visibilityTimeout int64

	msgInputChan    chan<- *sqs.Message
	msgToDeleteChan <-chan *sqs.Message

	closeSignal chan struct{}
	doneSignal  chan struct{}

	runErr error
}

func runInCtx(ctx context.Context, toRuns ...func(context.Context) error) error {
	runningWG := sync.WaitGroup{}
	closableContext, closeCallback := context.WithCancel(ctx)
	sentErrs := make(chan error, len(toRuns)+1)
	l := getLog(ctx)
	runningWG.Add(len(toRuns))
	for _, toRun := range toRuns {
		go func(toRun func(context.Context) error) {
			defer runningWG.Done()
			defer func() {
				p := recover()
				if p != nil {
					closeCallback()
					panic(p)
				}
			}()
			if err := toRun(closableContext); err != nil {
				sentErrs <- err
				closeCallback()
			}
		}(toRun)
	}
	l.Printf("Waiting for wg to finish")
	runningWG.Wait()
	close(sentErrs)
	select {
	case err := <-sentErrs:
		return err
	default:
		return nil
	}
}

func (q *queuePoller) Start(ctx context.Context) error {
	if q.closeSignal != nil {
		panic("Cannot start twice!")
	}

	q.closeSignal = make(chan struct{})
	q.doneSignal = make(chan struct{})

	go func() {
		q.runErr = runInCtx(ctx, q.removeMessages, q.drainMessages)
		close(q.doneSignal)
	}()
	return nil
}

func (q *queuePoller) Close() error {
	close(q.closeSignal)
	<-q.Done()
	return q.Err()
}

func (q *queuePoller) Done() <-chan struct{} {
	return q.doneSignal
}

func (q *queuePoller) Err() error {
	return q.runErr
}

func (q *queuePoller) removeMessages(ctx context.Context) error {
	service := sqs.New(q.cfg)
	for {
		select {
		case <-q.closeSignal:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case msgToRemove, ok := <-q.msgToDeleteChan:
			if !ok {
				return nil
			}
			input := sqs.DeleteMessageInput{
				QueueUrl:      &q.queueURL,
				ReceiptHandle: msgToRemove.ReceiptHandle,
			}
			out, err := service.DeleteMessage(&input)
			if err != nil {
				return wraperr(err, "unable to delete a message")
			}
			q.msgRemoveLog.Printf("%s\n", out.GoString())
		}
	}
}

func (q *queuePoller) forwardMsgs(ctx context.Context, resp *sqs.ReceiveMessageOutput) error {
	l := getLog(ctx)
	if len(resp.Messages) == 0 {
		l.Printf("Got no messages")
		return nil
	}
	for _, m := range resp.Messages {
		l.Printf("Message: %s", *m.MessageId)
		select {
		case <-q.closeSignal:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case q.msgInputChan <- m:
		}
	}
	return nil
}

func (q *queuePoller) drainMessages(ctx context.Context) error {
	service := sqs.New(q.cfg)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-q.closeSignal:
			return nil
		default:
		}
		msg := sqs.ReceiveMessageInput{
			QueueUrl:        &q.queueURL,
			WaitTimeSeconds: &q.waitTimeSeconds,
		}
		if q.visibilityTimeout != 0 {
			msg.VisibilityTimeout = &q.visibilityTimeout
		}
		resp, err := service.ReceiveMessage(&msg)
		if err != nil {
			return wraperr(err, "cannot receive messagges from queue")
		}
		if err := q.forwardMsgs(ctx, resp); err != nil {
			return err
		}
	}
}
