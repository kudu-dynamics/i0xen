package producer

import (
	"reflect"
	"time"

	log "github.com/sirupsen/logrus"
)

type JsonBlob = map[string]interface{}

type Producer interface {
	ProduceWork() (JsonBlob, error)

	Start() error

	ValidateWork(JsonBlob) (JsonBlob, error)
}

func StartProducers(producers []Producer) error {
	for _, p := range producers {
		err := p.Start()
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"producer": reflect.TypeOf(p),
			}).Error("could not start producer")
			return err
		}
	}
	return nil
}

func ProduceWork(producers []Producer) JsonBlob {
	// Query each of the producers in order to see if there is any work
	// available. If there is, attempt to validate the work unit before passing
	// it along to a waiting consumer.

	for _, p := range producers {
		work, err := p.ProduceWork()
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"producer": reflect.TypeOf(p),
			}).Error("could not produce work")
			continue
		}
		if work == nil {
			log.WithFields(log.Fields{
				"producer": reflect.TypeOf(p),
			}).Debug("no work to produce")
			continue
		}

		work, err = p.ValidateWork(work)
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"producer": reflect.TypeOf(p),
			}).Error("could not validate work")
			continue
		}

		log.WithFields(log.Fields{
			"work":     work,
			"producer": reflect.TypeOf(p),
		}).Debug("produced work")
		return work
	}
	// No available work. Sleep for a bit.
	time.Sleep(1 * time.Second)
	return nil
}
