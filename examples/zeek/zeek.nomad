# Run this from the `examples/zeek` directory.

locals {
  files = [
    "config.hcl",
    "entrypoint.sh",
    "scripts/install-zeek.sh",
    "scripts/pcap-to-zeek.sh",
  ]
}

job "analytic/i0xen/pcap-to-zeek" {
  datacenters = ["dc1"]
  type        = "service"
  priority    = 11

  meta {
    docker_image   = "ubuntu"
    docker_version = "18.04"
  }

  group "analytic/i0xen" {
    task "pcap-to-zeek" {
      artifact {
        source = "${AWS_ENDPOINT_URL}/public/tools/i0xen/i0xen.tar.gz"
      }

      driver = "docker"

      config {
        image        = "${NOMAD_META_docker_image}:${NOMAD_META_docker_version}"
        entrypoint   = ["/local/entrypoint.sh"]
        network_mode = "host"
        work_dir     = "${NOMAD_TASK_DIR}"
      }

      env {
        AWS_ENDPOINT_URL     = "${AWS_ENDPOINT_URL}"
      }

      dynamic "template" {
        for_each = local.files

        content {
          data        = file(template.value)
          destination = "/local/${template.value}"
          perms       = "755"
        }
      }
    }
  }
}
