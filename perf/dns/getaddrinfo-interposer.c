/* getaddrinfo interposer */

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <stdio.h>
#include <stdlib.h>
#include <dlfcn.h>
#include <errno.h>
#include <string.h>
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

/* defaults to 0.0.0.0. */
static in_addr_t proxy_ipv4_addr;

static __attribute__((constructor (101))) void setup(void)
{
	char *proxy_ipaddr = getenv("PROXY_IP");

	if ((libc_getaddrinfo = dlsym(RTLD_NEXT, "getaddrinfo")) == NULL) {
		fprintf(stderr, "error: dlsym(getaddrinfo) failed: %s\n", strerror(errno));
		exit(EXIT_FAILURE); /* has to be fatal */
	}

	/*
	 * If PROXY_IP is not in the environment and you use this
	 * interposer library then proxy_ipv4_addr will be 0.0.0.0
	 * (i.e., current host).
	 */
	if (proxy_ipaddr != NULL) {
		proxy_ipv4_addr = inet_addr(proxy_ipaddr);
	}

	return;
}

static int str_starts_with(const char *restrict string, const char *restrict prefix)
{
	while (*prefix) {
		if (*prefix++ != *string++) {
			return 0;
		}
	}
	return 1;
}

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

/* libc interposer */
int getaddrinfo(const char *node,
		const char *service,
		const struct addrinfo *hints,
		struct addrinfo **res)
{
	assert(libc_getaddrinfo != NULL);

	if (str_starts_with(node, "perf-test-hydra-")) {
		int rc = libc_getaddrinfo("127.0.0.1", service, hints, res);
		if (rc == 0) {
			/*
			 * Modify the result so any lookup with the
			 * prefix "perf-test-hydra-" will be replaced
			 * with the binary representation of
			 * $PROXY_IP.
			 */
			memcpy(&((struct sockaddr_in *)res[0]->ai_addr)->sin_addr,
			       &proxy_ipv4_addr, sizeof(proxy_ipv4_addr));
		}
		return rc;
	} else {
		return libc_getaddrinfo(node, service, hints, res);
	}
}
