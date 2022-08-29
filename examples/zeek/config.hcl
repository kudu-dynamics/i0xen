log_level = "info"

job {
  cmd           = "./scripts/pcap-to-zeek.sh"
  ext           = ".dns.log.gz"
  meta_required = ["download_url", "s3_output"]
  meta_optional = []
  name          = "pcap-to-zeek"
  payload       = "forbidden"
  s3_bucket     = "dnslogs"
  version       = 1
}

nats {
  subject = "test"
}
