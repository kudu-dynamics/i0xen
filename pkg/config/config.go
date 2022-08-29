package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Job      JobConfig
	LogLevel string `mapstructure:"log_level"`
	Nats     NatsConfig
}

type NatsConfig struct {
	QueueGroup string
	Subject    string
	Url        string
}

type JobConfig struct {
	Cmd          string
	OutputBucket string   `mapstructure:"s3_bucket"`
	OutputExt    string   `mapstructure:"ext"`
	MetaRequired []string `mapstructure:"meta_required"`
	MetaOptional []string `mapstructure:"meta_optional"`
	Name         string
	Payload      string
	Version      string
}

func Usage() string {
	return ""
}

func Load(config_file string) (cfg Config, err error) {
	v := viper.New()
	if config_file != "" {
		v.SetConfigFile(config_file)
	} else {
		v.SetConfigName("config.hcl")
		v.SetConfigType("hcl")
	}
	v.AddConfigPath("/local/")
	v.AddConfigPath(".")
	v.SetDefault("log_level", "info")

	// Configuration file takes precedence over environment variables.
	v.AutomaticEnv()
	err = v.ReadInConfig()
	if err != nil {
		return cfg, err
	}
	v.Unmarshal(&cfg, GetHclOverride())
	return cfg, nil
}

func (cfg Config) Validate() error {
	// Validate settings
	var errb strings.Builder
	sval := reflect.ValueOf(&cfg).Elem()
	stype := sval.Type()
	for i := 0; i < sval.NumField(); i++ {
		vf := sval.Field(i)
		tf := stype.Field(i)
		// Skip any fields that don't map to an environment variable.
		envvar := tf.Tag.Get("mapstructure")
		if envvar == "" {
			continue
		}
		// Skip any fields that are not strings.
		if vf.Kind() != reflect.String {
			continue
		}
		// Skip any fields that are already set.
		if vf.Interface() != "" {
			continue
		}
		// Skip any fields that are optional.
		if ot := tf.Tag.Get("optional"); ot != "" {
			continue
		}
		// Any fields that remain, cause errors when unset.
		fmt.Fprintf(&errb, "\nenvvar %s must be set", strings.ToUpper(envvar))
	}
	if errb.Len() > 0 {
		return errors.New(errb.String())
	}
	return nil
}

func (cfg Config) SetupLogging() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetOutput(os.Stderr)

	switch strings.ToLower(cfg.LogLevel) {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		// Unknown log level. Defaulting to INFO.
		fmt.Fprintf(os.Stderr, "defaulting log level to INFO\n")
		log.SetLevel(log.InfoLevel)
	}
}
