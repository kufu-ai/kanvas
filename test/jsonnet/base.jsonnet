{
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
}
