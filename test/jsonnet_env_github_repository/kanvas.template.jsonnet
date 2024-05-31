{
    components: {
        product1: {
            appimage: {
                dir: "/containerimages/app",
                docker: {
                    image: "myregistry/" + std.extVar("github_repo_owner") + "-" + std.extVar("github_repo_name"),
                }
            }
        }
    }
}
