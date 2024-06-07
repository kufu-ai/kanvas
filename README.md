# kanvas

`kanvas` is a high-level, go-to tool for deploying your container-based applications along with the whole infrastructure.

Tired of maintaining adhoc scripts or proprietary CI pipelines?

`kanvas`'s sole goal is to never require you to write an adhoc script to deploy your application. You write a kanvas configuration file once, and use it for various use-cases listed below...

## Use-cases

**Self-service isolated preview environments**: `kanvas` is ideal for providing self-service preview or deveploment environment. That is, every developer in your company can just run `kanvas apply` to get his own development/preview environment that contains all the VPC-related stuffs, ArgoCD, Kubernetes clusters, loadbalancers, and so on.

**Automate production deployments**: `kanvas` scales. Once you've satisfied with local tests, generate a proprietary CI config like one for GitHub Actions. It is able to run multiple Docker and Terraform jobs as concurrent as possible according to the DAG of the comonents leveraing the CI-specific concurrency primitive, like workflow jobs for GitHub Actions.

**Standardize how you deploy applications**: The examples contained in the `kanvas` project mainly targets EKS on AWS with ArgoCD. However, you can add any Terraform configs for ECS-based deployments or even GCP, Azure deployments.

## How it works

 `kanvas` enables you to declare plan-and-apply style deployment workflows that is powered by `Docker` and `Terraform`. You no longer need to write adhoc scripts, CircleCI(future) or GitHub Actions workflows to automate deployments.

## Getting started

`kanvas` is easy to get started. You don't need to reimplement your Docker and Terraform based deployment workflows using a completely different config management systems like Pulumi, AWS CDK, and so on.

Add `kanvas.yaml` to your project and write paths to directories and declare dependencies among directories, and you are good to go.

Under the hood, it runs on vanilla `docker` and `terraform`. Anything that can be deployed using the two should benefit from using `kanvas`.

`kanvas` is easy to test. Add `kanvas.yaml` to your config repository that contains `Dockerfile` and Terraform projects. Run `kanvas diff` to preview diffs and then run `kanvas apply` to deploy to the test environment.

## Configuration syntax

A kanvas configuration is conventionally written in a file named `kanvas.yaml`.

It must contain a `components` field that contains one or more components:

```yaml
components:
  infra:
    # snip
  k8s-cluster:
    # snip
  containerimage:
    # snip
  app:
    # snip
```

Each component requires `dir`, which is the path to the directory that contains everything you need to `plan` and `apply` the component:

```yaml
components:
  infra:
    dir: path/to/your/infra/terraform/project
  k8s-cluster:
    dir: path/to/your/k8s/terraform/project
  containerimage:
    dir: path/to/docker/build/context
  app:
    dir: path/to/k8s/manifests/or/docker-compose-config
```

Each component can optionally have `needs`, which lists names of other components that ths component depends on. Here, if we say `k8s-cluste depends on (or needs) infra`, that means the `infra` terraform project must be applied before the `k8s-cluster` terraform project is applied:

```yaml
components:
  infra:
    dir: path/to/your/infra/terraform/project
  k8s-cluster:
    dir: path/to/your/k8s/terraform/project
    needs:
    - infra
  containerimage:
    dir: path/to/docker/build/context
  app:
    dir: path/to/k8s/manifests/or/docker-compose-config
    needs:
    - k8s-cluster
    - containerimage
```

Each component takes any of the below provider configurations:

- `docker` provider is used to let kanvas build/tag/push a container image using the `docker` command.

  It's only supported setting is `image`, which is used to specify the whole image name except the tag suffix (which is sha256 of the `dir` contents).

  ```yaml
  containerimage:
    dir: path/to/docker/build/context
    docker:
      image: repo/image:tagprefix-
  ```
- `terraform` provider us used to let kanvas run terraform plan/apply to diff/deploy your infrastructure.

  It supports two options, `target` and `vars`.

  `target` is used to specify the plan/apply target a.k.a `-t (--target) $target` flag of the `terraform` command.

  `vars` is used to specify terraform vars for plan/apply a.k.a `-var name=$value` of the `terraform plan/apply` commands.

  ```yaml
  infra:
    dir: path/to/your/infra/terraform/project
  k8s-cluster:
    dir: path/to/your/k8s/terraform/project
    needs:
    - infra
    terraform:
      target: null_resource.infra
      vars:
      - name: vpc_id
        valueFrom: infra.vpc_id
  ```
- `kubernetes` deploys the k8s app in various ways using [kargo](https://github.com/mumoshu/kargo), with per-environment configuration for the right balance between speed and safety.

  The `kubernetes` provider can deploy your contained application to the K8s cluster by directly running
  `helm`, `kustomize`, `kompose`, or indirectly via `argocd`.

  By combining it with `environments`, it's easy to
  directly deploy for test and preview environments, while
  requiring deployments via ArgoCD for production for compliance and auditing.
  Please see [Environments](#advanced-environments) for more information.

  The below is the reference configuration that covers all the required and optional fields available for this provider:

  ```yaml
  app:
    # If dir contains `docker-compose.yml`, it runs `vals eval` and
    # `kompose` to transfom it to K8s manifests
    # If it contains `Chart.yaml`, it runs `helm diff` and `helm upgrade --install`.
    dir: deploy
    kubernetes:
      # This maps to --plugin-env in case you're going to uses the `argocd` option below.
      # Otherwise all the envs are set before calling commands (like kompose, kustomize, kubectl, helm, etc.)
      env:
      - name: STAGE
        value: prod
      - name: FOO
        valueFrom: component_name.foo
      # kustomize instructs kanvas to deploy the app using `kustomize`.
      # It has two major modes. The first mode directly calls `kustomize`, whereas
      # the second indirectly call it via `argocd`.
      # The first mode is triggered by setting only `helm`.
      # The second is enabled when you set `argocd` along with `kustomize`.
      kustomize:
        # kustomize.image maps to --kustomize-image of argocd-app-create.
        image:
      # helm instructs kanvas to deploy the app using `helm`.
      # It has two major modes. The first mode directly calls `helm`, whereas
      # the second indirectly call it via `argocd`.
      # The first mode is triggered by setting only `helm`.
      # The second is enabled when you set `argocd` along with `helm`.
      helm:
        # helm.repo maps to --repo of argocd-app-create
        # in case kubernetes.argocd is not empty.
        repo: https://charts.helm.sh/stable
        # --helm-chart
        chart: mychart
        # --revision
        version: 1.2.3
        # helm.set corresponds to `--helm-set $name=$value` flags of `argocd app create` command
        set:
        - name: foo
          value: foo
        - name: bar
          valueFrom: component_name.bar
      argocd:
        # argocd.repo maps to --repo of argocd-app-create.
        repo: github.com/myorg/myrepo.git
        # argocd.path maps to --path of argocd-app-create.
        # Note: In case you had kubernetes.dir along with argocd.path,
        # kanvas automatically git-push the content of dir to $argocd_repo/$argocd_path.
        # To opt-out of it, set `push: false`.
        path: path/to/dir/in/repo
        # --dir-recurse
        dirRecurse: true
        # --dest-namespace
        namespace: default
        # serverFrom maps to --dest-server where the flag value is take from the output of another kanvas component
        serverFrom: component_name.k8s_endpoint
        # Note that the config management plugin definition in the configmap
        # and the --config-management-plugin flag passed to argocd-app-create # command is auto-generated.
  ```

  Note that in case `$dir/kargo.yaml` exists, everything read from that yaml file becomes the default values.

  In other words, `kanvas` merges and overrdes settings provided via the component onto settings declared in the kargo config file.

  This enables you to "offload" some or all of the `kubernetes` provider settings to an external `kargo.yaml` file for clarity.

Here's a more complete example of `kanvas.yaml` that covers everything introduced so far:

```yaml
components:
  product1:
    components:
      appimage:
        dir: /containerimages/app
        docker:
          image: "davinci-std/example:myownprefix-"
      base:
        dir: /tf2
        needs:
        - appimage
        terraform:
          target: null_resource.eks_cluster
          vars:
          - name: containerimage_name
            valueFrom: appimage.id
      argocd:
        dir: /tf2
        needs:
        - base
        terraform:
          target: aws_alb.argocd_api
          vars:
          - name: cluster_endpoint
            valueFrom: base.cluster_endpoint
          - name: cluster_token
            valueFrom: base.cluster_token
      argocd_resources:
        # only path relative to where the command has run is supported
        # maybe the top-project-level "dir" might be supported in the future
        # which in combination with the relative path support for sub-project dir might be handy for DRY
        dir: /tf2
        needs:
        - argocd
        terraform:
          target: argocd_application.kanvas
```

In this example, the directory structure is supposed to be:

- product1/
  - base
  - argocd
  - argocd-resources

or

- product1/
  - 01-base
  - 02-argocd
  - 03-argocd-resources

In the second form, `needs` values are induced from the two digits before `-`, so that the projects are planned and applied in the ascending order of the denoted number.

The above configuration implies that:

- `kanvas plan` runs `terraform plan` and store the plan files up until the first unapplied terraform projects
- `kanvas apply` runs `docker build` and `terraform apply` for all the terraform projects planned beforehand

### Advanced: Synthetic Test

You can optionally add an `tests` field for defining two or more
tests. Each test is triggered after the prerequisites(components) are applied.

The whole `kanvas apply` run fails when any of the tests failed.

```
components:
  infra:
    dir: tf
    terraform:
      target: null_resource.infra

tests:
  pinglb:
    prober: icmp
    targetFrom: infra.alb_endpoint
```

Under the hood, kanvas runs [blackbox-exporter](https://github.com/prometheus/blackbox_exporter) with the provided test configuration.

### Advanced: Environments

You can optionally add an `environments` field for defining two or more environments.

Each environment can have `defaults`, `needs`, and `approval` fields.

All in all, this feature serves the following use-cases:

- [Keep multi-environment deployment config DRY (`defaults`)](#multi-env-deployment-using-the-environment-defaults)
- [Promote releases across the environments (`needs` and `approval`)](#promoting-releases-using-the-environment-needs)

#### Multi-env deployment using the environment "defaults"

Each environment can have a "defaults" field for setting default values used for every component defined in the kanvas.yaml file. This helps making your multi-environment configuration DRY.

Why?

Let's say you have two "terraform" components in the config file and you want to use the "production" terraform workspace for production deployments while using the "preview" workspace for preview environments.

Without environments, you need to have two kanvas.yaml files and two pipelines/workflows/jobs//etc to run kanvas for respectively environments.

With environments, you can literally specify which terraform is used across all the terraform components per environment:

`kanvas.yaml`:

```
environments:
  production:
    defaults:
      terraform:
        workspace: production
  preview:
    defaults:
      terraform:
        workspace: preview

components:
  infra:
    dir: tf
    terraform:
      target: null_resource.infra
  k8s:
    needs:
    - infra
    dir: tf
    terraform:
      target: null_resource.k8s
```

Compare this is wrinting and maintaining two kanvas.yaml files like:

`production/kanvas.yaml`:

```
components:
  infra:
    dir: tf
    terraform:
      target: null_resource.infra
      workspace: production
  k8s:
    needs:
    - infra
    dir: tf
    terraform:
      target: null_resource.k8s
      workspace: production
```

`preview/kanvas.yaml`:

```
components:
  infra:
    dir: tf
    terraform:
      target: null_resource.infra
      workspace: preview
  k8s:
    needs:
    - infra
    dir: tf
    terraform:
      target: null_resource.k8s
      workspace: preview
```

Although the total LOC does not differ much, the version with environments have two benefits.

First, you no longer need to repeat "workspace" fields for all the components. This is especially nice when there are more components than environments, which might be common.

Second, you no longer need to duplicate whole components across environments. The more components you have, the nicer it becomes.

#### Promoting releases using the environment "needs"

Each environment can be configured to be deployed only after another envionment(s).

This is useful when you want some environment to be planned and applied only after other environments are successfully applied.

An example use-case for this would be to ensure changes working on the preview environment before you apply the changes to the production environment.

To make an environment (say "production") depend on another (say "preview"), add an `needs` field under the environment and specify dependent environment names.

```
environments:
  production:
    needs:
    - preview
  preview: {}
```

The field is intentionally given the same name as the component `needs` field, to make it clear that this is the standard way to denote dependencies for anything in `kanvas`.

We will support a few strategies to implement this:

- If you run `kanvas apply` locally, it will just apply those environments in the order of dependencies.
- If export it to GitHub Actions, it gives you two options.
  - The first option is to have a single pull request workflow to `kanvas apply` the environments, with intermediate ["manual approval"](https://trstringer.com/github-actions-manual-approval/) steps.
  - The second option is to have two workflows, one for deploying the `preview` environment on each pull request, and another for deploying the `production` environment on push to the main branch. Note though, this will work only for two environments only.

#### Advanced Example

A more complete example of the whole configuration which involves `environments`, `defaults`, and `needs` are shown below for your reference.

```
environments:
  production:
    after:
    - preview
    defaults:
      terraform:
        # This sets terraform.workspace for all the terraform components,
        # "infra" and "k8s" in this example.
        workspace: prod
  preview:
    defaults:
      terraform:
        # This sets terraform.workspace for all the terraform components,
        # "infra" and "k8s" in this example.
        workspace: prev

components:
  image:
    dir: dockerfile
    docker:
      image: "davinci-std/example:myownprefix-"
    # We opt to use te specific image repository only for the preview environments:
      preview:
        docker:
          image: "myrepo/example:myownprefix-"
  infra:
    dir: tf
    terraform:
      target: null_resource.infra
  k8s:
    needs:
    - infra
    dir: tf
    terraform:
      target: null_resource.k8s
  app:
    needs:
    - k8s
    - image
```

## FAQ

- Why not use `terraform apply -target` for multi-phase terraform apply?
  That's because `terraform` itself wanrs not to use it routinely. Here's the
  excerpt from the terraform warning message that tells it:

  ```
  Note that the -target option is not suitable for routine use, and is
  │ provided only for exceptional situations such as recovering from errors or
  │ mistakes, or when Terraform specifically suggests to use it as part of an
  │ error message.
  ```

  `kanvas`, on the other hand, is build from ground-up to natively connect
  various jobs powered by `docker`, `terraform`, `argocd`, `kubectl`, etc and
  there is no technical limit of sticking with `-target` for applying a set
  terraform configurations in sequence. You don't need to rely on `-target`,
  nor aggregate many terraform resources into a single Terraform project
  for the sake of completeness and clarity.

  Instead, do split your Terraform projects into layers, and write a single
  `kanvas.yaml` for glueing them up.

## Roadmap

Here's the preliminary list of roadmap items to be implemented:

- [ ] Ability to specify the Terraform project templates for reuse
- [ ] Ability to export the workflow to CodeBuild (Multiple kanvas jobs are mapped to a single build)
- [ ] Ability to export the workflow to GitHub Actions (Each kanvas job is mapped to one Actions job)
- [ ] An example project that covers kompose, EKS, and ArgoCD.
  - We'll be using [vals](https://github.com/helmfile/vals) for embedding secret references in `docker-compose.yml`, [kompose](https://kompose.io/) for converting `docker-compose.yml` to Kubernetes manifests, apply or git-push the manifests somehow(perhaps we'll be creating a dedicated tool for that), and [terraform-argocd-provider](https://github.com/oboukili/terraform-provider-argocd) for managing ArgoCD projects, applications, cluster secrets, and so on.

## Related projects

`kanvas` is supposed to be a spritual successor to [Atlantis](https://github.com/runatlantis/atlantis). Like Atlantis, we needed something that is convenient to power mainly-Terraform-based workflows. We also needed out-of-box support for more tools, wanted it be easier to operate and easier to test locally and easier to integrate into various environments.

## References

We've carefully surveyed the current state of Terraform and its wrappers in the opensource community and found something like `kanvas` is missing and here we are.

- https://github.com/hashicorp/terraform/issues/25244
- https://github.com/hashicorp/terraform/issues/30937
- https://github.com/hashicorp/terraform/issues/4149
- https://discuss.hashicorp.com/t/multiple-plan-apply-stages/8320
- https://developer.hashicorp.com/terraform/language/state/remote-state-data
- https://registry.terraform.io/providers/hashicorp/tfe/latest/docs/data-sources/outputs
- https://terragrunt.gruntwork.io/docs/features/work-with-multiple-aws-accounts/ terragrunt is great but scales vertically
- https://github.com/transcend-io/terragrunt-atlantis-config generates atlantis config for terragrunt. We generate GitHub Actions workflow config for our own terraform+alpha wrapper
- https://terragrunt.gruntwork.io/docs/features/execute-terraform-commands-on-multiple-modules-at-once/#dependencies-between-modules terragrunt manages dependencies amongn terraform projects. We manage dependencies across any supported jobs (currently docker and terraform)

