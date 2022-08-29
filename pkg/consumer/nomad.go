package consumer

// Call the user-specified command.
//
// Populate the environment variables with meta variables in line with the
// Nomad runtime environment.
//
// - https://www.nomadproject.io/docs/runtime/environment
//
// XXX: Need to clear any sensitive environment variables that the parent
//      might be carrying.

import (
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"syscall"

	"github.com/kudu/i0xen/config"
	. "github.com/kudu/i0xen/producer"
	log "github.com/sirupsen/logrus"
)

type NomadConsumer struct {
	cfg config.Config
}

func NewNomadConsumer(cfg config.Config) *NomadConsumer {
	return &NomadConsumer{
		cfg,
	}
}

func (n *NomadConsumer) Start() error {
	// This consumer is stateless.
	return nil
}

func (n *NomadConsumer) ConsumeWork(wg *sync.WaitGroup, work JsonBlob) {
	defer wg.Done()

	log.WithFields(log.Fields{
		"work":     work,
		"consumer": reflect.TypeOf(n),
	}).Info("consuming work")

	err := n.inferior(work)
	if err != nil {
	}
}

func (n *NomadConsumer) inferior(work JsonBlob) (err error) {
	// Copy all of the environment variables of the host and pass them through
	// to the inferior process while also adding the rewritten NOMAD_META
	// variables.
	meta := work["meta"].(JsonBlob)
	env := os.Environ()
	for k, v := range meta {
		env = append(
			env,
			fmt.Sprintf(
				"NOMAD_META_%v=%v",
				strings.ToLower(k),
				v,
			),
		)
	}

	cmd := exec.Command(n.cfg.Job.Cmd)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set a signal for child processes to receive when the parent dies.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	return cmd.Run()
}
