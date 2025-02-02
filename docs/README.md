# Documentation

This document gives an overview and simple examples about the usage of the component-cli.
For a comprehensive documentation see the [generated docs](./reference/component-cli.md).

__Index__:

- [Credential Handling](#credential-handling)
- [Resource](#create-a-component)
  - [Add a resource](#add-a-resource)
  - [Add a local blob](#add-a-local-file)
  - [Use simple templating](#use-simple-templating)
- [ComponentReference](#add-a-dependency-to-a-component)
- [Remote](#remote)
  - [Push a Component Descriptor](#push)

## Credential Handling

The credentials for accessing private registries are handled via plain Docker CLI mechanisms. Either use the `docker login` command, or edit the Docker config file on your local machine manually.

Specialties regarding credential handling in the component-cli:

1) The component-cli uses credential entries from the `auths` section in preference over `credHelpers` for the same registry URL.

```json
{
	"auths": {
		"eu.gcr.io": {}
	},
	"credHelpers": {
		"eu.gcr.io": "gcloud",
	},
	...
}
```

With the above Docker config, the component-cli will use the set of credentials from the `auths` section when interacting with the `eu.gcr.io` registry.

2) The component-cli supports providing credentials for subpaths of the same host (not supported by Docker CLI).

```json
{
	"auths": {
		"eu.gcr.io/my-project": {},
    "eu.gcr.io": {}
	},
	...
}
```

With the above Docker config, the component-cli will use the more specific set of credentials for all artifacts under the path `eu.gcr.io/my-project`.

## Create a Component

A component can be created by using the `create` subcommand.

```shell script
# create a directory to work with
$ mkdir ./examples
$ component-cli ca create ./examples --component-name "example.com/component/name" --component-version "v0.0.1"
```

## Add a Resource

A resource can be added to an existing component descriptor by using the `resource` subcommand.

The subcommand adds all resources defined in by a file.

```shell script
# define by file
$ cat <<EOF > ./resource.yaml
name: 'ubuntu'
version: 'v0.0.1'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
EOF
$ component-cli ca resources add ./examples ./resource.yaml
```

The resources can also be added using stdin.

```shell script
# define by file
$ cat <<EOF | component-cli ca resources add ./examples -
name: 'ubuntu'
version: 'v0.0.1'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
EOF
```

The file is expected to be a yaml, and multiple resources can be added by using the yaml multi doc syntax:

```yaml
---
name: 'myconfig'
type: 'json'
relation: 'local'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:18.0'
...
```

### add a local file

A local blob (any file) can be added to the component descriptor by using the `input` attribute.
This will automatically add the file as local artifact to the component descriptor and will configure the access of the resource accordingly.

:warning: Note you can specify that the given blob is automatically gzipped by setting the `compress` attribute.<br>
:warning: When the given input path is a directory, a tar archive is automatically created.

```shell script
$ cat <<EOF > ./blob-resource.yaml
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "./blob.raw" # path is realtive to the current resource file
  mediaType: "application/x-elf" # optional, will be defaulted to application/octet-stream
EOF

$ cat <<EOF > ./blob.raw
{
  "key": "value"
}
EOF

$ component-cli ca resources add ./examples ./blob-resource.yaml
```

See an example with a directory and possible options.

```yaml
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "some/path"
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "dir"
  path: /my/path
  compress: true # defaults to false
  exclude: "*.txt"
...
```

## Add a dependency to a component

A component reference can be added to an existing component descriptor by using the `component-reference` subcommand.

The subcommand offers the possibility to add a component-reference either via component-reference template in a file or by defining the component-reference via commandline flags.<br>
:warning: Note that the commandline flags overwrite values defined by the file.

```shell script
# define by file
$ cat <<EOF > ./comp-ref.yaml
---
name: 'cli'
componentName: 'github.com/gardener/component-spec'
version: 'v0.0.1'
extraIdentity:
  myid: abc
  mysecondid: efg
labels:
- name: mylabel
  values: efg
- name: mysecondlabel
  value:
    key: true
EOF
$ component-cli ca component-reference add ./examples ./comp-ref.yaml
```

## use simple templating

With the component-cli resources, sources and component-references can be dynamically added to a component descriptor.
Often these definitions need to be templated with the current build values like the version.

One solution for that issue is to do the templating yourself, with your preferred templating engine.
This approach is also recommended when you need a more advanced templating features like loops.

In most use cases, a simple variable substitution is enough to meet the requirements.
Therefore, the component-cli offers the possibility to use simple variable expansion in the templates.

For example if a resources need to be templates with a new version, the resource definition would be defined as follows:

```yaml
name: 'ubuntu'
version: '${VERSION}'
type: 'ociImage'
relation: 'external'
access:
  type: 'ociRegistry'
  imageReference: 'ubuntu:${VERSION}'
```

With the command `component-cli ca resource add [path to component archive] [myfile] -- VARIABLE=v0.0.2` it is now possible to define key-value pairs for the substitution.

## Remote

The `remote` subcommand contains utility functions to interact with component references stored in a remote component repository (oci repository).

:warning: Currently the component-cli uses the default docker config for authentication.
Use `docker login` to authenticate against a oci repository.

## Push

A component descriptor in component archive CTF format can be pushed to an component repository (oci repository) by using the `remote push` command.

This command takes 1 argument which is the path to the component archive.<br>
A component archive is a directory that contains the component descriptor at `/component-descriptor.yaml` and all blobs at `/blobs/<blobname>`.

```shell script
$ component-cli ca remote push [path to component archive]
```
