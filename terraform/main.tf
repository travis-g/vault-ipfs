variable "ipfs_version" {
  default = "latest"
}

variable "ipfs_instance_count" {
  default = 1
}
variable "ipfs_storage_max" {}

resource "docker_image" "ipfs" {
  name         = "ipfs/go-ipfs:${var.ipfs_version}"
  keep_locally = true
}

data "template_file" "ipfs_config" {
  count    = "${var.ipfs_instance_count}"
  template = "${file("${path.module}/templates/ipfs_config.tpl")}"

  vars {
    ipfs_api_port     = "${format("50%d1", count.index)}"
    ipfs_swarm_port   = "${format("40%d1", count.index)}"
    ipfs_gateway_port = "${format("808%d", count.index)}"
    ipfs_storage_max  = "${var.ipfs_storage_max}"
  }
}

resource "docker_container" "ipfs_server" {
  count = "${var.ipfs_instance_count}"
  name  = "ipfs-${format("ipfs%d", count.index)}"
  image = "${docker_image.ipfs.latest}"

  env = [
    "IPFS_LOGGING=info",
  ]

  command  = ["daemon", "--migrate=true"]
  hostname = "${format("ipfs%d", count.index)}"
  must_run = true

  volumes {
    host_path      = "${path.module}/../ipfs/ipfs${count.index}"
    container_path = "/ipfs/data"
  }

  upload {
    content = "${element(data.template_file.ipfs_config.*.rendered, count.index)}"
    file    = "/ipfs/data/config"
  }

  # Daemon API, do not expose publicly
  ports {
    internal = "5001"
    external = "${format("50%d1", count.index)}"
    protocol = "tcp"
  }

  # Swarm TCP
  ports {
    internal = "4001"
    external = "${format("40%d1", count.index)}"
    protocol = "tcp"
  }

  # Web Gateway
  ports {
    internal = "8080"
    external = "${format("808%d", count.index)}"
    protocol = "tcp"
  }

  # Swarm WebSockets (TODO: overlaps)
  ports {
    internal = "8081"
    external = "8081"
    protocol = "tcp"
  }
}
