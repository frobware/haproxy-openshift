/* getaddrinfo interposer */

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <stdio.h>
#include <stdlib.h>
#include <dlfcn.h>
#include <errno.h>
#include <string.h>
#include <unistd.h>
#include <limits.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netdb.h>
#include <arpa/inet.h>

/* interposed functions. */
static int (*libc_getaddrinfo)(const char *node,
			       const char *service,
			       const struct addrinfo *hints,
			       struct addrinfo **res);
static char *proxy_host;
static char hostname[HOST_NAME_MAX+1];

#define BACKEND_PREFIX "perf-test-hydra"

static int string_starts_with(const char *string, const char *prefix)
{
	while (*prefix) {
		if (*prefix++ != *string++) {
			return 0;
		}
	}
	return 1;
}

static __attribute__((constructor)) void setup()
{
	if ((libc_getaddrinfo = dlsym(RTLD_NEXT, "getaddrinfo")) == NULL) {
		fprintf(stderr, "error: dlsym(getaddrinfo) failed: %s\n", strerror(errno));
		exit(EXIT_FAILURE); /* has to be fatal */
	}

	if (gethostname(hostname, HOST_NAME_MAX) != 0) {
		fprintf(stderr, "error: cannot determine hostname: %s\n", strerror(errno));
		exit(EXIT_FAILURE); /* has to be fatal */
	}

	proxy_host = getenv("PROXY_HOST");
	if (proxy_host == NULL || *proxy_host == '\0') {
		proxy_host = hostname;
	}

	fprintf(stderr, "getaddrinfo(): lookups with prefix '%s' will resolve as '%s'\n",
		BACKEND_PREFIX,
		proxy_host);
}

/* libc interposer */
int getaddrinfo(const char *node,
		const char *service,
		const struct addrinfo *hints,
		struct addrinfo **res)
{
	if (string_starts_with(node, BACKEND_PREFIX)) {
		return libc_getaddrinfo(proxy_host, service, hints, res);
	} else {
		return libc_getaddrinfo(node, service, hints, res);
	}
}
