variable "name" {
    type = string
}

output "cluster_endpoint" {
  value = "example.com:1234"
}

output "cluster_token" {
  value = "mytoken"
}

variable "vpc_name" {
    type = string
}

resource "null_resource" "eks_cluster" {
  triggers = {
  } 
}
