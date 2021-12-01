# Augment openshift-router image with new haproxy binary

The scripts in this directory facilitate building a new version of
`haproxy` and inserting that new binary into an existing
openshift-router image.

# Install

### CLone this repo

	$ git clone https://github.com/frobware/haproxy-openshift
	$ cd haproxy-openshift

### Build UBI7 and UBI8 images

	$ ./toolbox-create-build-containers.sh

	$ podman images |grep haproxy-builder
	localhost/haproxy-builder-ubi8                           latest                             80888392c2c2  54 minutes ago     604 MB
	localhost/haproxy-builder-ubi7                           latest                             9a2d845bc124  55 minutes ago     499 MB

The script will also create two new containers and they are used by
the build script to build haproxy from source:

	$ toolbox list
	IMAGE ID      IMAGE NAME                  CREATED
	43d9dec0fe5e  localhost/ubi7-aim:latest   2 hours ago
	fccd3ffa629c  localhost/ubi8-aim:latest   2 hours ago

	CONTAINER ID  CONTAINER NAME        CREATED         STATUS      IMAGE NAME
	488e74f64bc9  haproxy-builder-ubi7  9 minutes ago   running     localhost/haproxy-builder-ubi7:latest
	86e3086f8fa1  haproxy-builder-ubi8  10 minutes ago  configured  localhost/haproxy-builder-ubi8:latest

Creating these images and containers is a one-off exercise. The script
`./toolbox-create-build-containers.sh` only needs to be rerun if you
delete the containers or images.

# Example workflow

	$ git clone https://github.com/frobware/haproxy-openshift

## OpenShift v3

	$ git clone http://git.haproxy.org/git/haproxy-1.8.git/
	$ cd haproxy-1.8
	$ ../haproxy-openshift/build-image.sh --build-container haproxy-builder-ubi7 --build-script ../haproxy-openshift/build-haproxy-1.8.sh -f ../haproxy-openshift/Dockerfile.3.11 --push-image --dry-run
	toolbox run --container haproxy-builder-ubi7 ../haproxy-openshift/build-haproxy-1.8.sh
	podman build -t amcdermo/openshift-router:ocp-3.11-haproxy-v1.8.28 -f ../haproxy-openshift/Dockerfile.3.11 .
	podman tag amcdermo/openshift-router:ocp-3.11-haproxy-v1.8.28 quay.io/amcdermo/openshift-router:ocp-3.11-haproxy-v1.8.28
	podman push quay.io/amcdermo/openshift-router:ocp-3.11-haproxy-v1.8.28

The toolbox container (in this case the ubi7 version) is used to build
a new `haproxy` binary from source. The container build then copies
the local `./haproxy` binary into the image; see the `COPY` command in
the various container files.:

```Dockerfile
COPY ./haproxy /usr/sbin/haproxy
```

Finally the image is tagged and can now be pushed. Once pushed you can
update the openshift deployment to refer to this new image spec.

## OpenShift v4.8 - haproxy-2.2

	$ git clone http://git.haproxy.org/git/haproxy-2.2.git/
	$ cd haproxy-2.2
	$ ../haproxy-openshift/build-image.sh --build-container haproxy-builder-ubi8 --build-script ../haproxy-openshift/build-haproxy-2.2.sh -f ../haproxy-openshift/Dockerfile.4.8 --push-image --dry-run
	toolbox run --container haproxy-builder-ubi8 ../haproxy-openshift/build-haproxy-2.2.sh
	podman build -t amcdermo/openshift-router:ocp-4.8-haproxy-v2.2.17 -f ../haproxy-openshift/Dockerfile.4.8 .
	podman tag amcdermo/openshift-router:ocp-4.8-haproxy-v2.2.17 quay.io/amcdermo/openshift-router:ocp-4.8-haproxy-v2.2.17

## OpenShift v4.10 - haproxy-2.4

	$ git clone http://git.haproxy.org/git/haproxy-2.4.git/
	$ cd haproxy-2.4
	$ ../haproxy-openshift/build-image.sh --build-container haproxy-builder-ubi8 --build-script ../haproxy-openshift/build-haproxy-2.4.sh -f ../haproxy-openshift/Dockerfile.4.10 --push-image --dry-run
	toolbox run --container haproxy-builder-ubi8 ../haproxy-openshift/build-haproxy-2.4.sh
	podman build -t amcdermo/openshift-router:ocp-4.10-haproxy-v2.4.9 -f ../haproxy-openshift/Dockerfile.4.10 .
	podman tag amcdermo/openshift-router:ocp-4.10-haproxy-v2.4.9 quay.io/amcdermo/openshift-router:ocp-4.10-haproxy-v2.4.9
	podman push quay.io/amcdermo/openshift-router:ocp-4.10-haproxy-v2.4.9

## Overriding various names

The names that are used for the registry, the registry username, and
the image name can all be customised by specifying them explicitly
when invoking `./build-image.sh`.

For example:

	$ REGISTRY=docker.io REGISTRY_USERNAME=frobware IMAGENAME=openshift-router-perfscale \
	  ../haproxy-openshift/build-image.sh \
		--build-container haproxy-builder-ubi8 \
		--build-script ../haproxy-openshift/build-haproxy-2.4.sh \
		--containerfile ../haproxy-openshift/Dockerfile.4.10 \
		--push-image \
		--dry-run
	toolbox run --container haproxy-builder-ubi8 ../haproxy-openshift/build-haproxy-2.4.sh
	podman build -t frobware/openshift-router-perfscale:ocp-4.10-haproxy-v2.4.9 -f ../haproxy-openshift/Dockerfile.4.10 .
	podman tag frobware/openshift-router-perfscale:ocp-4.10-haproxy-v2.4.9 docker.io/frobware/openshift-router-perfscale:ocp-4.10-haproxy-v2.4.9
	podman push docker.io/frobware/openshift-router-perfscale:ocp-4.10-haproxy-v2.4.9

## Injecting a pre-built binary

If you already have a pre-built `haproxy` binary (e.g., brew build /
RPM file) then you can inject it directly:

    $ cp /path/to/pre-built/haproxy .
	$ TAGNAME=ocp-4.10-haproxy-v2.2.19 REGISTRY_USERNAME=amcdermo IMAGENAME=openshift-router-perfscale \
		./build-image.sh \
			--build-container haproxy-builder-ubi8 \
			--build-script /bin/true \
			--containerfile Dockerfile.4.10 \
			--push-image \
			--dry-run
	toolbox run --container haproxy-builder-ubi8 /bin/true
	podman build -t amcdermo/openshift-router-perfscale:ocp-4.10-haproxy-v2.2.19 -f Dockerfile.4.10 .
	podman tag amcdermo/openshift-router-perfscale:ocp-4.10-haproxy-v2.2.19 quay.io/amcdermo/openshift-router-perfscale:ocp-4.10-haproxy-v2.2.19
	podman push quay.io/amcdermo/openshift-router-perfscale:ocp-4.10-haproxy-v2.2.19
