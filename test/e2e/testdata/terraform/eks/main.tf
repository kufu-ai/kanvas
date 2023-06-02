# https://docs.aws.amazon.com/eks/latest/userguide/troubleshooting.html

variable "name" {
    type = string
}

variable "region" {
    type = string
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }

    local = {
      source = "hashicorp/local"
      version = "~> 2.1.0"
    }

    null = {
      source = "hashicorp/null"
      version = "~> 3.1.0"
    }
  }
}

variable "default_tags" {
  default     = {
     Environment = "test"
     Owner       = "mumoshu"
     Project     = "test"
  }
}

provider "aws" {
  # default_tags {
  #   tags = {
  #     Environment = "test"
  #     Owner       = "mumoshu"
  #     Project     = "test"
  #   }
  # }
  region = var.region
}

data "aws_caller_identity" "self" {

}

data "aws_region" "current" {
}

locals {
  aws_account_id              = data.aws_caller_identity.self.account_id
  region                      = data.aws_region.current.name
  containerimage_name         = "kanvastestmumoshu"
  containerimage_buildcontext = "../containerimages/app"
}

resource "aws_eks_cluster" "example" {
  name     = var.name
  role_arn = aws_iam_role.cluster.arn

  vpc_config {
    # subnet_ids = [aws_subnet.example1.id, aws_subnet.example2.id]
    subnet_ids = aws_subnet.private[*].id
  }

  enabled_cluster_log_types = ["api", "audit"]

  # Ensure that IAM Role permissions are created before and deleted after EKS Cluster handling.
  # Otherwise, EKS will not be able to properly delete EKS managed EC2 infrastructure such as Security Groups.
  depends_on = [
    aws_iam_role_policy_attachment.cluster-AmazonEKSClusterPolicy,
    aws_iam_role_policy_attachment.cluster-AmazonEKSVPCResourceController,
    aws_cloudwatch_log_group.example,
  ]

  tags = merge(
    var.default_tags,
  )
}

resource "aws_cloudwatch_log_group" "example" {
  # The log group name format is /aws/eks/<cluster-name>/cluster
  # Reference: https://docs.aws.amazon.com/eks/latest/userguide/control-plane-logs.html
  name              = "/aws/eks/${var.name}/cluster"
  retention_in_days = 1

  tags = merge(
    var.default_tags,
  )
}

resource "aws_iam_role" "cluster" {
  name = "eks-cluster-${var.name}"

  assume_role_policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "eks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
POLICY

  tags = merge(
    var.default_tags,
  )
}

resource "aws_iam_role_policy_attachment" "cluster-AmazonEKSClusterPolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
  role       = aws_iam_role.cluster.name
}

resource "aws_iam_role_policy_attachment" "cluster-AmazonEKSVPCResourceController" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSVPCResourceController"
  role       = aws_iam_role.cluster.name
}

output "endpoint" {
  value = aws_eks_cluster.example.endpoint
}

output "kubeconfig-certificate-authority-data" {
  value = aws_eks_cluster.example.certificate_authority[0].data
}

data "aws_availability_zones" "available" {
  state = "available"
  //{
  // RespMetadata: {
  //  StatusCode: 400,
  //  RequestID: "4133f45d-3f1a-425f-ac53-ee520f47648c"
  //},
  //ClusterName: "test1",
  //Message_: "Cannot create cluster 'test1' because ap-northeast-1a, the targeted availability zone, does not currently have sufficient capacity to support the cluster. Retry and choose from these availability zones: ap-northeast-1b, ap-northeast-1c, ap-northeast-1d",
  //ValidZones: ["ap-northeast-1b","ap-northeast-1c","ap-northeast-1d"]
  //}
  exclude_zone_ids = ["apne1-az3"]
}

variable "vpc_name" {
    type = string
}

# resource "aws_vpc_ipam" "example" {
#   operating_regions {
#     region_name = data.aws_region.current.name
#   }
# }

# resource "aws_vpc_ipam_pool" "example" {
#   address_family = "ipv6"

#   ipam_scope_id  = aws_vpc_ipam.example.public_default_scope_id
#   locale         = data.aws_region.current.name
#   aws_service    = "ec2"
# }

resource "aws_vpc" "example" {
  assign_generated_ipv6_cidr_block = true
  cidr_block = "10.1.0.0/16"
  // Computed attributes cannot be set, but a value was set for "ipv6_cidr_block".
  # ipv6_cidr_block = local.vpc_ipv6_cidr_block
  # ipv6_ipam_pool_id = aws_vpc_ipam_pool.example.id

  tags = merge(
    var.default_tags,
    {
      Name = var.vpc_name
    },
  )
}

// required by ngw
resource "aws_internet_gateway" "example" {
  vpc_id = aws_vpc.example.id

  tags = merge(
    var.default_tags,
  )
}

resource "aws_eip" "ngw" {
  # otherwise allocation_id is missing and you end up with the following error on nat gateway:
  #   The argument "allocation_id" is required, but no definition was found.
  vpc = true

  # tags cannot be set for a standard-domain EIP - must be a VPC-domain EIP
  #tags = {...}
}

// required for nodes to join
// also see https://docs.aws.amazon.com/eks/latest/userguide/troubleshooting.html
resource "aws_nat_gateway" "ngw" {
  allocation_id = aws_eip.ngw.id
  subnet_id     = aws_subnet.public[0].id
  depends_on = [aws_internet_gateway.example]

  tags = merge(
    var.default_tags,
  )
}

locals {
  public_subnet_count = 2
  private_subnet_count = 2
  vpc_ipv6_cidr_block = "2406:da14:9ba:b500::/56"
  public_ipv6_cidr_block = [
    "2406:da14:9ba:b510::/64",
    "2406:da14:9ba:b511::/64",
  ]
  private_ipv6_cidr_block = [
    "2406:da14:9ba:b500::/64",
    "2406:da14:9ba:b501::/64",
  ]
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.example.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.example.id
  }

  tags = merge(
    var.default_tags,
  )
}

resource "aws_route_table_association" "public" {
  count = local.public_subnet_count
  subnet_id = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public.id
}

resource "aws_subnet" "public" {
  count = local.public_subnet_count

  availability_zone = data.aws_availability_zones.available.names[count.index]
  # cidr_block        = cidrsubnet(aws_vpc.example.cidr_block, 8, count.index + 10)
  cidr_block        = cidrsubnet(aws_vpc.example.cidr_block, 4, count.index)
  vpc_id            = aws_vpc.example.id
  map_public_ip_on_launch = true
  # ipv6_cidr_block = local.public_ipv6_cidr_block[count.index]
  ipv6_cidr_block   = cidrsubnet(aws_vpc.example.ipv6_cidr_block, 8, count.index)
  assign_ipv6_address_on_creation = true

  tags = merge(
    var.default_tags,
    {
      Name = "${var.vpc_name}-public-${count.index}"
      "kubernetes.io/cluster/${var.name}" = "shared"
    },
  )
}

resource "aws_route_table" "private" {
  vpc_id = aws_vpc.example.id
  route {
    cidr_block = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.ngw.id
  }

  tags = merge(
    var.default_tags,
  )
}

resource "aws_route_table_association" "private" {
  count = local.private_subnet_count
  subnet_id = aws_subnet.private[count.index].id
  route_table_id = aws_route_table.private.id
}

resource "aws_subnet" "private" {
  count = local.private_subnet_count

  availability_zone = data.aws_availability_zones.available.names[count.index]
  # cidr_block        = cidrsubnet(aws_vpc.example.cidr_block, 8, count.index)
  vpc_id            = aws_vpc.example.id
  # ipv6_cidr_block = local.private_ipv6_cidr_block[count.index]

  // We add the number of public networks to the netnum to differentiate
  // CIRDs among public and private subnets.
  // See https://zenn.dev/shonansurvivors/articles/5424c50f5fd13d
  cidr_block        = cidrsubnet(aws_vpc.example.cidr_block, 4, count.index + length(aws_subnet.public))
  // We use the /64 subnet for IPv6
  // See https://medium.com/@mattias.holmlund/setting-up-ipv6-on-amazon-with-terraform-e14b3bfef577
  ipv6_cidr_block   = cidrsubnet(aws_vpc.example.ipv6_cidr_block, 8, count.index + length(aws_subnet.public))
  

  tags = merge(
    var.default_tags,
    {
      Name = "${var.vpc_name}-private-${count.index}"
      "kubernetes.io/cluster/${var.name}" = "shared"
    },
  )
}

//
// nodegroups
//

resource "aws_eks_node_group" "stateless1" {
  cluster_name    = aws_eks_cluster.example.name
  node_group_name = "stateless1"
  node_role_arn   = aws_iam_role.nodegroup.arn
  subnet_ids      = aws_subnet.public[*].id
  disk_size       = 100

  scaling_config {
    desired_size = 2
    max_size     = 3
    min_size     = 1
  }

  # lifecycle {
  #   ignore_changes = [scaling_config[0].desired_size]
  # }

  # Ensure that IAM Role permissions are created before and deleted after EKS Node Group handling.
  # Otherwise, EKS will not be able to properly delete EC2 Instances and Elastic Network Interfaces.
  depends_on = [
    aws_nat_gateway.ngw,
    aws_iam_role_policy_attachment.nodegroup-AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.nodegroup-AmazonEKS_CNI_Policy,
    aws_iam_role_policy_attachment.nodegroup-AmazonEC2ContainerRegistryReadOnly,
  ]

  tags = merge(
    var.default_tags,
  )
}

resource "aws_eks_node_group" "stateless2" {
  cluster_name    = aws_eks_cluster.example.name
  node_group_name = "stateless2"
  node_role_arn   = aws_iam_role.nodegroup.arn
  subnet_ids      = aws_subnet.public[*].id

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  lifecycle {
    ignore_changes = [scaling_config[0].desired_size]
  }

  # launch_template {
  #   id = aws_launch_template.lc.id
  #   version = aws_launch_template.lc.latest_version
  # }

  # Ensure that IAM Role permissions are created before and deleted after EKS Node Group handling.
  # Otherwise, EKS will not be able to properly delete EC2 Instances and Elastic Network Interfaces.
  depends_on = [
    aws_nat_gateway.ngw,
    aws_iam_role_policy_attachment.nodegroup-AmazonEKSWorkerNodePolicy,
    aws_iam_role_policy_attachment.nodegroup-AmazonEKS_CNI_Policy,
    aws_iam_role_policy_attachment.nodegroup-AmazonEC2ContainerRegistryReadOnly,
  ]

  tags = merge(
    var.default_tags,
  )
}

locals {
  eks-node-private-userdata = <<USERDATA
MIME-Version: 1.0
Content-Type: multipart/mixed; boundary="==MYBOUNDARY=="

--==MYBOUNDARY==
Content-Type: text/x-shellscript; charset="us-ascii"

#!/bin/bash -xe
# See https://github.com/weaveworks/eksctl/issues/3572#issuecomment-855328193
sudo mkdir -p /etc/containerd
containerd config default | sed "/containerd.runtimes.runc.options/a SystemdCgroup = true" | sudo tee /etc/containerd/config.toml
sudo systemctl restart containerd
cat /etc/kubernetes/kubelet/kubelet-config.json | jq ".cgroupDriver |= \"systemd\"" > /tmp/kubelet-config.json
sudo mv /tmp/kubelet-config.json /etc/kubernetes/kubelet/kubelet-config.json
sudo sed -i -e "s#--container-runtime\ docker#--container-runtime\ remote --container-runtime-endpoint\ unix:///run/containerd/containerd.sock#" /etc/systemd/system/kubelet.service
sudo systemctl daemon-reload
sudo systemctl restart kubelet.service
VERSION="v1.20.0"; wget --no-verbose https://github.com/kubernetes-sigs/cri-tools/releases/download/$VERSION/crictl-$VERSION-linux-amd64.tar.gz -O /tmp/crictl-linux-amd64.tar.gz
sudo tar zxvf /tmp/crictl-linux-amd64.tar.gz -C /usr/bin
cat <<EOS | sudo tee /etc/crictl.yaml
runtime-endpoint: unix:///run/containerd/containerd.sock
EOS
# See https://github.com/aws-samples/terraform-eks-code/blob/master/nodeg/user_data.tf
sudo /etc/eks/bootstrap.sh --apiserver-endpoint '${aws_eks_cluster.example.endpoint}' --b64-cluster-ca '${aws_eks_cluster.example.certificate_authority[0].data}' '${aws_eks_cluster.example.name}'
echo "Running custom user data script" > /tmp/me.txt
yum install -y amazon-ssm-agent
echo "yum'd agent" >> /tmp/me.txt
systemctl enable amazon-ssm-agent && systemctl start amazon-ssm-agent
date >> /tmp/me.txt

--==MYBOUNDARY==--
USERDATA

  key_name = var.name

  allowed_hosts = ["${chomp(data.http.ip.body)}/32"]
}

data "http" "ip" {
  url = "http://ipv4.icanhazip.com"
}

locals {
  public_key_file  = "./.key_pair/${local.key_name}.id_rsa.pub"
  private_key_file = "./.key_pair/${local.key_name}.id_rsa"
}

resource "tls_private_key" "main" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "local_file" "private_key_pem" {
  filename = local.private_key_file
  content  = tls_private_key.main.private_key_pem
  file_permission = "0600"
  provisioner "local-exec" {
    command = "chmod 600 ${local.private_key_file} && chown 1000:1000 ${local.private_key_file}"
  }
}

resource "local_file" "public_key_openssh" {
  filename = local.public_key_file
  content  = tls_private_key.main.public_key_openssh
  provisioner "local-exec" {
    command = "chmod 600 ${local.public_key_file}"
  }
}

resource "aws_key_pair" "main" {
  key_name   = local.key_name
  public_key = tls_private_key.main.public_key_openssh
}

data "aws_ssm_parameter" "eksami" {
  name = format("/aws/service/eks/optimized-ami/%s/amazon-linux-2/recommended/image_id", aws_eks_cluster.example.version)
}

resource "aws_launch_template" "lc" {
  instance_type           = "t3.medium"
  key_name                = aws_key_pair.main.id
  name                    = format("at-lt-%s-ng1", aws_eks_cluster.example.name)
  tags                    = {}
  image_id                = data.aws_ssm_parameter.eksami.value
  user_data               = base64encode(local.eks-node-private-userdata)
  vpc_security_group_ids  = [
    aws_eks_cluster.example.vpc_config[0].cluster_security_group_id,
    aws_security_group.nodes.id,
    # aws_security_group.lb_sg.id
  ]
  tag_specifications { 
    resource_type = "instance"
    tags = {
      Name = format("%s-ng1", aws_eks_cluster.example.name)
    }
  }
  lifecycle {
    create_before_destroy=true
  }
}

resource "aws_iam_role" "nodegroup" {
  name = "eks-cluster-${var.name}-nodegroup"

  assume_role_policy = jsonencode({
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ec2.amazonaws.com"
      }
    }]
    Version = "2012-10-17"
  })

  tags = merge(
    var.default_tags,
  )
}

resource "aws_iam_role_policy_attachment" "nodegroup-AmazonEKSWorkerNodePolicy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
  role       = aws_iam_role.nodegroup.name
}

resource "aws_iam_role_policy_attachment" "nodegroup-AmazonEKS_CNI_Policy" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
  role       = aws_iam_role.nodegroup.name
}

resource "aws_iam_role_policy_attachment" "nodegroup-AmazonEC2ContainerRegistryReadOnly" {
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
  role       = aws_iam_role.nodegroup.name
}

resource "aws_iam_role_policy_attachment" "nodegroup-AmazonEC2RoleforSSM" {
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2RoleforSSM"
  role       = aws_iam_role.nodegroup.name
}

//
// Add more attachments folllowing https://github.com/aws-samples/terraform-eks-code/tree/master/iam
//

resource "aws_security_group" "bastion" {
  name        = "Bastion host of EKS cluster ${aws_eks_cluster.example.name}"
  description = "Allow SSH access to bastion host and outbound internet access"
  vpc_id      = aws_vpc.example.id

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    
  }
}

resource "aws_security_group_rule" "ssh" {
  protocol          = "TCP"
  from_port         = 22
  to_port           = 22
  type              = "ingress"
  cidr_blocks       = local.allowed_hosts
  security_group_id = aws_security_group.bastion.id
}

resource "aws_security_group_rule" "internet" {
  protocol          = "-1"
  from_port         = 0
  to_port           = 0
  type              = "egress"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.bastion.id
}

resource "aws_security_group" "nodes" {
  name        = "Nodes of EKS cluster ${aws_eks_cluster.example.name}"
  description = "Allow SSH access from bastion"
  vpc_id      = aws_vpc.example.id

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    
  }
}

resource "aws_security_group_rule" "intranet" {
  protocol          = "TCP"
  from_port         = 22
  to_port           = 22
  type              = "ingress"
  source_security_group_id = aws_security_group.bastion.id
  security_group_id = aws_security_group.nodes.id
}


resource "aws_security_group_rule" "nodes_internet_egress" {
  protocol          = "-1"
  from_port         = 0
  to_port           = 0
  type              = "egress"
  cidr_blocks       = ["0.0.0.0/0"]
  security_group_id = aws_security_group.nodes.id
}

resource "aws_instance" "bastion" {
  ami                         = data.aws_ssm_parameter.eksami.value
  instance_type               = "t3.micro"
  key_name                    = aws_key_pair.main.id
  subnet_id                   = aws_subnet.public[0].id
  vpc_security_group_ids      = [aws_security_group.bastion.id]
  associate_public_ip_address = true

  root_block_device {
    volume_size           = 10
    delete_on_termination = true
  }

  lifecycle {
    ignore_changes = [ami]
  }

  tags = {
    Name    = "bastion"
  }
}

resource "null_resource" "bastion" {
  triggers = {
    ssh_key = local_file.private_key_pem.id
    bastion = aws_instance.bastion.id
    ssh = aws_security_group_rule.ssh.id
    internet = aws_security_group_rule.internet.id
  }
}

resource "local_file" "ca" {
    content     = base64decode(aws_eks_cluster.example.certificate_authority.0.data)
    filename = "${path.module}/ca"
}

data "aws_eks_cluster" "example" {
  # name = "example"
  name = aws_eks_cluster.example.name
}

data "aws_eks_cluster_auth" "example" {
  # name = "example"
  name = aws_eks_cluster.example.name
}
