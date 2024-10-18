name: QuickStart example

# retrieve latest provider release
sources:
  githubrelease:
    kind: githubrelease
    name: Get the latest provider release
    spec:
      owner: "rancher-sandbox"
      repository: "cluster-api-provider-azure"
      token: "{{ .github.token }}"
      username: "salasberryfin"
      typeFilter:
        latest: true

# first check if given release is available
conditions:
  FileExists:
    name: |
      Is release 'rancher-sandbox/cluster-api-provider-azure:{{ source `githubrelease` }} available?
    kind: file
    disablesourceinput: true
    spec:
      file: https://github.com/rancher-sandbox/cluster-api-provider-azure/releases/download/{{ source `githubrelease` }}/infrastructure-components.yaml

# update config.yaml accordingly
targets:
  bumpprovider:
    name: Bump provider version
    kind: file
    sourceid: githubrelease # Will be ignored as `replacepattern` is specified
    spec:
      file: data.yaml
      matchpattern: 'https:\/\/github\.com\/rancher-sandbox\/cluster-api-provider-azure\/releases\/v[0-9]+\.[0-9]+\.[0-9]+\/infrastructure-components\.yaml'
      replacepattern: 'https://github.com/rancher-sandbox/cluster-api-provider-azure/releases/{{ source `githubrelease` }}/infrastructure-components.yaml'

# create a pr with the changes
actions:
  default:
    title: Open a GitHub pull request with new updates
    kind: github/pullrequest
    scmID: default
    target:
      - bumpprovider
    spec:
      automerge: false
      mergemethod: squash
      labels:
        - provider
