resource "null_resource" "argocd" {
  triggers = {
  }
}

variable "cluster_endpoint" {
  type = string
}

variable "cluster_token" {
  type = string
}

output "argocd_server" {
  # Ensure you run:
  #   kubectl port-forward svc/argocd-server -n argocd 8080:443
  value = "localhost:8080"
}

output "argocd_username" {
  value = "admin"
}

output "argocd_password" {
 # If you're testing this locally, install argocd like:
 #  helm upgrade -n argocd --install \
 #    argocd \
 #    argo/argo-cd \
 #    --set configs.secret.argocdServerAdminPassword=$(htpasswd -bnBC 10 "" mypassword | tr -d ':\n')
 value = "mypassword"
}

output "argocd_insecure" {
  value = true
}
