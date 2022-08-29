package consumer

import (
	"reflect"
	"sync"

	. "github.com/kudu/i0xen/producer"
	log "github.com/sirupsen/logrus"
)

type Consumer interface {
	ConsumeWork(*sync.WaitGroup, JsonBlob)

	Start() error
}

func StartConsumers(consumers []Consumer) error {
	for _, c := range consumers {
		err := c.Start()
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"consumer": reflect.TypeOf(c),
			}).Error("could not start consumer")
			return err
		}
	}
	return nil
}

func ConsumeWork(consumers []Consumer, work JsonBlob) {
	// Give all consumers the same unit of work and wait for them to finish.
	var wg sync.WaitGroup
	for _, c := range consumers {
		wg.Add(1)
		go c.ConsumeWork(&wg, work)
	}
	wg.Wait()
}
