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

#undef NDEBUG /* always fire on assert */
#include <assert.h>

/* interposed functions. */
static int (*libc_getaddrinfo)(const char *node,
			       const char *service,
			       const struct addrinfo *hints,
			       struct addrinfo **res);
static char *proxy_host;
static char hostname[HOST_NAME_MAX+1];

#if 0
static void print_v4_addr(in_addr_t ipv4_addr)
{
	char addrbuf[INET_ADDRSTRLEN + 1];
	const char *addr;
	addr = inet_ntop(AF_INET, &ipv4_addr, addrbuf, sizeof addrbuf);
	if (addr == NULL) {
		fprintf(stderr, "IPv4 %s\n", strerror(errno));
		abort();
	}
	fprintf(stderr, "IPv4: %s\n", addr);
}

static void print_addrinfo(struct addrinfo *list)
{
	struct addrinfo *curr;

	for (curr = list; curr != NULL; curr = curr->ai_next) {
		fprintf(stderr, "curr = %p, next = %p, addrlen = %ld, flags = %d, protocol = %d, family = %d, socktype = %d ", curr, curr->ai_next, (long)curr->ai_addrlen, curr->ai_flags, curr->ai_protocol, curr->ai_family, curr->ai_socktype);
		if (curr->ai_family == AF_INET) {
			char addrbuf[INET_ADDRSTRLEN + 1];
			const char *addr;
			addr = inet_ntop(AF_INET, &(((struct sockaddr_in *)curr->ai_addr)->sin_addr), addrbuf, sizeof addrbuf);
			if (addr == NULL) {
				fprintf(stderr, "IPv4 %s\n", strerror(errno));
				abort();
			}
			fprintf(stderr, "IPv4: %s\n", addr);
		} else if (curr->ai_family == AF_INET6) {
			char addrbuf[INET6_ADDRSTRLEN + 1];
			const char *addr;
			addr = inet_ntop(AF_INET6, &(((struct sockaddr_in6 *)curr->ai_addr)->sin6_addr), addrbuf, sizeof addrbuf);
			if (addr == NULL) {
				fprintf(stderr, "IPv6 %s.\n", strerror(errno));
				abort();
			}
			fprintf(stderr, "IPv6: %s\n", addr);
		}
	}
}
#endif

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

	printf("proxy_host %s\n", proxy_host);
	fprintf(stderr, "Setting PROXY_HOST=%s\n", proxy_host);
}

/* libc interposer */
int getaddrinfo(const char *node,
		const char *service,
		const struct addrinfo *hints,
		struct addrinfo **res)
{
	if (string_starts_with(node, "perf-test-hydra-")) {
		return libc_getaddrinfo(proxy_host, service, hints, res);
	}
	return libc_getaddrinfo(node, service, hints, res);
}
