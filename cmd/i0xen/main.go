package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/kudu/i0xen/config"
	. "github.com/kudu/i0xen/consumer"
	. "github.com/kudu/i0xen/producer"
	"github.com/kudu/i0xen/version"
	log "github.com/sirupsen/logrus"
)

const Usage = `i0xen, a Nomad parameterized job supervisor.

Design goals are as follows:

- Bake extensions (e.g. metrics, tracing, etc.) into a sidecar rather than the
  application itself.

- Self-policing resource usage and limits in the presence of system monitors
  (e.g. an aggressive OOM killer) is unreliable. Therefore, move this
  responsibility to a sidecar service.

Usage:
  i0xen [options]

Options:
  -h --help           Show this screen.
  --version           Show version.
  -c --config=<file>  Configuration file to load.
`

type Ctx struct {
	consumers []Consumer

	// DEV: Multiple producers are possible.
	//
	//      In the future, it would be nice to listen for work on multiple NATS
	//      subjects that represent distinct priority or urgency levels.
	producers []Producer
}

func sighandler() {
	// Handle signals.

	log.Info("setting up signal handler")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGHUP)

	for {
		signal := <-sigChan
		if signal == os.Interrupt {
			log.Warn("good bye!")
			os.Exit(0)
		}

		if signal == syscall.SIGHUP {
			log.Warn("HUP!")
			os.Exit(0)
		}
	}
}

func main() {
	// Configure the application.
	opts, _ := docopt.ParseArgs(Usage, nil, version.GetVersion())
	config_file, _ := opts.String("--config")
	cfg, err := config.Load(config_file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	cfg.SetupLogging()

	// XXX: Hard-code selection of components.
	ctx := &Ctx{}
	ctx.consumers = []Consumer{NewNomadConsumer(cfg)}
	ctx.producers = []Producer{NewNatsProducer(cfg)}

	// Initialize components.
	log.Info("starting producers")
	for {
		err := StartProducers(ctx.producers)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	log.Info("starting consumers")
	for {
		err := StartConsumers(ctx.consumers)
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	// Main Loop.
	go sighandler()

	for {
		work := ProduceWork(ctx.producers)
		if work != nil {
			ConsumeWork(ctx.consumers, work)
		}
	}
}
