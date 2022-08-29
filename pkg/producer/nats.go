package producer

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	. "github.com/kudu/i0xen/config"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	log "github.com/sirupsen/logrus"
)

type NatsProducer struct {
	cfg        Config
	conn       *nats.EncodedConn
	next       chan JsonBlob
	subscribed bool
}

func NewNatsProducer(cfg Config) *NatsProducer {
	return &NatsProducer{
		cfg,
		nil,
		make(chan JsonBlob, 1),
		false,
	}
}

func (n *NatsProducer) Start() error {
	// Set defaults and validate settings.
	if n.cfg.Nats.QueueGroup == "" {
		n.cfg.Nats.QueueGroup = "i0xen"
	}
	if n.cfg.Nats.Subject == "" {
		log.WithFields(log.Fields{
			"producer": reflect.TypeOf(n),
		}).Fatal("invalid NATS subject: cannot be empty")
	}

	// Connect to the upstream NATS server.
	nc, err := nats.Options{
		AllowReconnect:    true,
		AsyncErrorCB:      n.asyncErrCB,
		DisconnectedErrCB: n.disconnectCB,
		ReconnectedCB:     n.reconnectCB,
		MaxReconnect:      -1,
		ReconnectWait:     5 * time.Second,
		SubChanLen:        1,
		Timeout:           1 * time.Second,
		Url:               n.cfg.Nats.Url,
	}.Connect()
	if err != nil {
		return err
	}

	nec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		return err
	}

	n.conn = nec
	return nil
}

func (n *NatsProducer) ProduceWork() (JsonBlob, error) {
	if !n.subscribed {
		// Fetch a single message from NATS.
		sub, err := n.conn.QueueSubscribe(
			n.cfg.Nats.Subject,
			n.cfg.Nats.QueueGroup,
			func(subject, reply string, body JsonBlob) {
				n.next <- JsonBlob{
					"subject": subject,
					"reply":   reply,
					"body":    body,
				}
			},
		)
		if err != nil {
			return nil, err
		}
		sub.AutoUnsubscribe(1)
		n.subscribed = true
	}

	// Check if there is any available work to process.
	select {
	case work := <-n.next:
		n.subscribed = false
		return work, nil
	default:
		return nil, nil
	}
}

func (n *NatsProducer) ValidateWork(msg JsonBlob) (JsonBlob, error) {
	// Add meta if missing.
	body := msg["body"].(JsonBlob)
	meta := add_meta_if_missing(body)

	// Add S3 sink.
	add_s3_sink(n.cfg.Job, meta)

	// Transform meta.
	whitelist := []string{}
	whitelist = append(whitelist, n.cfg.Job.MetaRequired...)
	whitelist = append(whitelist, n.cfg.Job.MetaOptional...)
	transform_meta(meta, whitelist)

	// Ensure required meta fields are present.
	err := ensure_required_meta(meta, n.cfg.Job.MetaRequired)

	// Reply with a payload if necessary.
	reply := msg["reply"].(string)
	if reply != "" {
		payload := JsonBlob{
			"timestamp": time.Now().UTC(),
		}
		if err != nil {
			payload["error"] = fmt.Sprintf("%v", err)
			payload["success"] = false
		} else {
			// DEV: The current ecosystem works with MinIO at the core.
			//      `s3_output` is the main expected key to coordinate work.
			if _, found := meta["s3_output"]; found {
				payload["s3_output"] = meta["s3_output"]
			}
			payload["success"] = true
		}
		// If we are unable to reply, drop the request entirely and wait
		// for the client to resubmit.
		err = n.conn.Publish(reply, payload)
	}
	return body, err
}

// NATS callback handlers.

func (n *NatsProducer) asyncErrCB(c *nats.Conn, sub *nats.Subscription, err error) {
	log.WithFields(log.Fields{
		"error":        err,
		"producer":     reflect.TypeOf(n),
		"subscription": sub,
	}).Warn("NATS subscription error")

	n.subscribed = false
}

func (n *NatsProducer) disconnectCB(c *nats.Conn, err error) {
	log.WithFields(log.Fields{
		"error":    err,
		"producer": reflect.TypeOf(n),
	}).Warn("disconnected from NATS")

	n.subscribed = false
}

func (n *NatsProducer) reconnectCB(c *nats.Conn) {
	log.WithFields(log.Fields{
		"producer": reflect.TypeOf(n),
	}).Warn("reconnected to NATS")
}

// Validation Utilities.

func add_meta_if_missing(body JsonBlob) JsonBlob {
	_, found := body["meta"]
	if !found {
		body["meta"] = make(JsonBlob)
	}
	return body["meta"].(JsonBlob)
}

func add_s3_sink(cfg JobConfig, meta JsonBlob) {
	// Format a random object path for clients and consumers to coordinate
	// storing results to.
	now := time.Now().UTC()
	uuidVal, _ := uuid.NewRandom()

	// XXX: Source these by configuration values.
	job_bucket := cfg.OutputBucket
	job_name := cfg.Name
	job_file_extension := strings.TrimLeft(".json", ".")
	job_version := cfg.Version

	object := strings.Join(
		[]string{
			uuidVal.String(),
			job_file_extension,
		},
		".",
	)
	meta["s3_output"] = strings.Join(
		[]string{
			job_bucket,
			job_name,
			job_version,
			fmt.Sprintf("%04d", now.Year()),
			fmt.Sprintf("%02d", now.Month()),
			fmt.Sprintf("%02d", now.Day()),
			object,
		},
		"/",
	)
}

func ensure_required_meta(meta JsonBlob, required []string) error {
	keys := []string{}
	for k := range meta {
		keys = append(keys, k)
	}

	missing := []string{}
	for _, param := range required {
		if !contains(keys, param) {
			missing = append(missing, param)
		}
	}
	if len(missing) > 0 {
		var errb strings.Builder
		fmt.Fprintf(&errb, "missing required meta parameters: ")
		for _, param := range missing {
			fmt.Fprintf(&errb, fmt.Sprintf("%v,", param))
		}
		return errors.New(strings.TrimRight(errb.String(), ","))
	}
	return nil
}

func transform_meta(meta JsonBlob, whitelist []string) {
	// The meta variables are destined to become environment variables for the
	// consumers. This requires that all variables become strings.
	for k, v := range meta {
		rt := reflect.TypeOf(v)
		switch rt.Kind() {
		case reflect.Array, reflect.Slice:
			// Lists need to be transformed into a single string value.
			newV := []string{}
			for _, v := range v.([]interface{}) {
				newV = append(newV, fmt.Sprintf("%v", v))
			}
			meta[k] = strings.Join(newV, ",")
		case reflect.String:
			// Strings can be ignored.
			break
		default:
			meta[k] = fmt.Sprintf("%v", v)
		}
	}

	// Drop all empty or unrecognized meta parameters.
	for k, v := range meta {
		if !contains(whitelist, k) {
			delete(meta, k)
			continue
		}

		if v == "" {
			delete(meta, k)
			continue
		}
	}
}

// Basic Utilities.

func contains(xs []string, s string) bool {
	for _, v := range xs {
		if v == s {
			return true
		}
	}
	return false
}
