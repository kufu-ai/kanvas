{
  "components": {
    "product1": {
      "components": {
        "appimage": {
          "dir": "/containerimages/app",
          "docker": {
            "image": "davinci-std/example:myownprefix-"
          }
        },
        "base": {
          "dir": "/tf2",
          "needs": [
            "appimage"
          ],
          "terraform": {
            "target": "null_resource.eks_cluster",
            "vars": [
              {
                "name": "containerimage_name",
                "valueFrom": "appimage.id"
              }
            ]
          }
        },
        "argocd": {
          "dir": "/tf2",
          "needs": [
            "base"
          ],
          "terraform": {
            "target": "aws_alb.argocd_api",
            "vars": [
              {
                "name": "cluster_endpoint",
                "valueFrom": "base.cluster_endpoint"
              },
              {
                "name": "cluster_token",
                "valueFrom": "base.cluster_token"
              }
            ]
          }
        },
        "argocd_resources": {
          "dir": "/tf2",
          "needs": [
            "argocd"
          ],
          "terraform": {
            "target": "argocd_application.kanvas"
          }
        }
      }
    }
  }
}
