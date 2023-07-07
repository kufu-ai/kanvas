local appimage = import 'appimage.jsonnet';

{
  "components": { // Define all the components here
    "product1": {
      "components": {
        "appimage": appimage,
        "base": import 'base.jsonnet',
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
