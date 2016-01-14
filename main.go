package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"golang.org/x/net/context"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
	"io"
	"net/url"
	"strconv"
)

type verboseLog uint32

const (
	verboseLogInstance verboseLog = iota
)

func setLog(ctx context.Context, log logger) context.Context {
	return context.WithValue(ctx, verboseLogInstance, log)
}

func getLog(ctx context.Context) logger {
	return ctx.Value(verboseLogInstance).(logger)
}

func wraperr(err error, msg string, args ...interface{}) *wrappedError {
	if err == nil {
		panic("You probably didn't mean to wrap a nil error")
	}
	return &wrappedError{
		err: err,
		msg: fmt.Sprintf(msg, args...),
	}
}

func logIfErr(l logger, err error, msg string, args ...interface{}) {
	if err != nil {
		l.Printf("%s: %s", err.Error(), fmt.Sprintf(msg, args...))
	}
}

type wrappedError struct {
	err error
	msg string
}

func (e *wrappedError) Error() string {
	return fmt.Sprintf("%s: %s", e.msg, e.err.Error())
}

type logger interface {
	Printf(string, ...interface{})
}

type goLogAwsLogger struct {
	*log.Logger
}

func (g goLogAwsLogger) Log(args ...interface{}) {
	g.Logger.Println(args...)
}

type buildTrigger struct {
	region            string
	queueURL          string
	apiToken          string
	circleToken       string
	phaburl           string
	visibilityTimeout int64
	verbose           bool
	verboseFile       string
	logOut            io.Writer
}

var mainInstance buildTrigger

func init() {
	flag.StringVar(&mainInstance.region, "region", os.Getenv("SQS_REGION"), "AWS region to send the message to")
	flag.StringVar(&mainInstance.queueURL, "queue", os.Getenv("SQS_QUEUE"), "SQS queue URL")
	flag.StringVar(&mainInstance.apiToken, "apitoken", os.Getenv("PHAB_API_TOKEN"), "Phabricator api token")
	flag.StringVar(&mainInstance.circleToken, "circletoken", os.Getenv("CIRCLECI_TOKEN"), "Token to use for CircleCI")
	flag.StringVar(&mainInstance.phaburl, "phaburl", "http://phabricator.corp.signalfx.com", "Phabricator URL")

	defaultVerbose, _ := strconv.ParseBool(os.Getenv("BUILD_VERBOSE"))
	flag.BoolVar(&mainInstance.verbose, "verbose", defaultVerbose, "Enable verbose logging")

	flag.StringVar(&mainInstance.verboseFile, "verbosefile", os.Getenv("BUILD_VERBOSE_FILE"), "File to put verbose logging into")

	defaultVisibility, _ := strconv.ParseInt(os.Getenv("QUEUE_VISIBILITY"), 10, 64)
	flag.Int64Var(&mainInstance.visibilityTimeout, "visibility", defaultVisibility, "If non zero, will change how long the message is hidden from other queue requests")
}

func main() {
	flag.Parse()
	exitOnErr(mainInstance.main(), os.Exit)
}

func exitOnErr(err error, osExit func(int)) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		osExit(1)
	}
}

func (c *buildTrigger) getAwsConfig(out io.Writer) *aws.Config {
	var logger aws.Logger
	if out != ioutil.Discard {
		logger = goLogAwsLogger{log.New(out, "[aws-go]", log.LstdFlags)}
	}
	return &aws.Config{
		Region: &c.region,
		Logger: logger,
	}
}

var errPleaseSpecifyRegion = errors.New("please specify a region")
var errPleaseSpecifyQueue = errors.New("please specify a queue URL")
var errPleaseSpecifyAPIToken = errors.New("please specify API token")

func (c *buildTrigger) parseFlags() error {
	c.logOut = ioutil.Discard
	if c.verbose {
		c.logOut = os.Stderr
	}
	if c.verboseFile != "" && c.verboseFile != "-" {
		c.logOut = &lumberjack.Logger{
			Filename:   c.verboseFile,
			MaxSize:    100,
			MaxBackups: 3,
		}
	}
	if c.region == "" {
		return errPleaseSpecifyRegion
	}
	if c.queueURL == "" {
		return errPleaseSpecifyQueue
	}
	if c.apiToken == "" {
		return errPleaseSpecifyAPIToken
	}
	if c.circleToken == "" {
		return errors.New("please specify a cirlce token")
	}
	return nil
}

func (c *buildTrigger) processParsedMessages(ctx context.Context, parsedMsgs chan parsedMessage, msgsFailedToProcess chan parsedMessage, msgToDeleteChan chan *sqs.Message, scriptLogger logger) error {
	for m := range parsedMsgs {
		if err := m.Execute(ctx); err != nil {
			scriptLogger.Printf("Error executing message: %s", err.Error())
			msgsFailedToProcess <- m
			continue
		}
		scriptLogger.Printf("Yay the message was processed correctly!  I should probably delete %s", *m.OriginalMsg().MessageId)
		msgToDeleteChan <- m.OriginalMsg()
	}
	return nil
}

func (c *buildTrigger) main() error {
	if err := c.parseFlags(); err != nil {
		return err
	}
	scriptLogger := log.New(c.logOut, "[buildtrigger]", log.LstdFlags)
	deleteMsgLogger := log.New(c.logOut, "[delete-msg]", log.LstdFlags)
	phabURL, err := url.Parse(c.phaburl)
	if err != nil {
		return wraperr(err, "cannot parse phab URL")
	}
	scriptLogger.Printf("Starting up")
	ctx := setLog(context.Background(), scriptLogger)
	cfg := c.getAwsConfig(c.logOut)
	ch := make(chan *sqs.Message)
	invalidMessages := make(chan *sqs.Message)
	msgsFailedToProcess := make(chan parsedMessage)
	parsedMsgs := make(chan parsedMessage)
	msgToDeleteChan := make(chan *sqs.Message)

	go func() {
		for m := range invalidMessages {
			scriptLogger.Printf("A message I don't know what to do with: %s", m.String())
			msgToDeleteChan <- m
		}
	}()

	go func() {
		for m := range msgsFailedToProcess {
			scriptLogger.Printf("A messaged failed to process.  Let's ignore it and try again? %s", *m.OriginalMsg().MessageId)
		}
	}()

	go c.processParsedMessages(ctx, parsedMsgs, msgsFailedToProcess, msgToDeleteChan, scriptLogger)

	tmpDir, err := ioutil.TempDir("", "buildtrigger")
	if err != nil {
		return wraperr(err, "cannot create temp directory for github publisher")
	}
	scriptLogger.Printf("Buliding inside directory %s", tmpDir)
	defer func() {
		logIfErr(scriptLogger, os.RemoveAll(tmpDir), "Cannote remove %s", tmpDir)
	}()

	phab := &phabricatorConduit{
		apiToken: c.apiToken,
		url:      phabURL,
	}

	gp := githubPusher{
		phab:   phab,
		tmpDir: tmpDir,
		cc: &circleClient{
			token: c.circleToken,
		},
	}

	cp := circleManager{
		git:  &gp,
		phab: phab,
		ci:   gp.cc,
	}

	hp := harbormasterPublisher{
		gp: &gp,
	}

	mp := newMsgProcessor(ch, invalidMessages, parsedMsgs, []msgConstructor{hp.parseHarbormasterMsg, cp.parseCircleCImsg})

	q := queuePoller{
		cfg:               cfg,
		queueURL:          c.queueURL,
		visibilityTimeout: c.visibilityTimeout,
		waitTimeSeconds:   20,
		msgRemoveLog:      deleteMsgLogger,

		msgInputChan:    ch,
		msgToDeleteChan: msgToDeleteChan,
	}
	if err := q.Start(ctx); err != nil {
		return err
	}
	if err := mp.Start(ctx); err != nil {
		return err
	}
	scriptLogger.Printf("Goroutines started")
	<-q.Done()
	if err := q.Err(); err != nil {
		return err
	}
	scriptLogger.Printf("q done")
	<-mp.Done()
	if err := mp.Err(); err != nil {
		return err
	}
	scriptLogger.Printf("mp done")
	return nil
}
