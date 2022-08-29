module i0xen

go 1.15

require (
	github.com/docopt/docopt-go v0.0.0-20180111231733-ee0de3bc6815
	github.com/kudu/i0xen/config v0.0.0-00010101000000-000000000000
	github.com/kudu/i0xen/consumer v0.0.0-00010101000000-000000000000
	github.com/kudu/i0xen/producer v0.0.0-00010101000000-000000000000
	github.com/kudu/i0xen/version v0.0.0-00010101000000-000000000000
	github.com/sirupsen/logrus v1.7.0
)

replace (
	github.com/kudu/i0xen/config => ./pkg/config
	github.com/kudu/i0xen/consumer => ./pkg/consumer
	github.com/kudu/i0xen/producer => ./pkg/producer
	github.com/kudu/i0xen/version => ./pkg/version
)
