/* getrandom interposer */

#ifndef _GNU_SOURCE
#define _GNU_SOURCE
#endif

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <dlfcn.h>
#include <errno.h>
#include <sys/random.h>

#undef NDEBUG /* always fire on assert */
#include <assert.h>

/* interposed functions. */
static ssize_t (*libc_getrandom)(void *buf, size_t buflen, unsigned int flags);

static __attribute__((constructor)) void setup()
{
	if ((libc_getrandom = dlsym(RTLD_NEXT, "getrandom")) == NULL) {
		fprintf(stderr, "error: dlsym(getrandom) failed: %s\n", strerror(errno));
		exit(EXIT_FAILURE); /* has to be fatal */
	}

	fprintf(stderr, "interposing getrandom()\n");
}

/* libc interposer */
ssize_t getrandom(void *buf, size_t buflen, unsigned int flags)
{
	memset(buf, 0x5e, buflen);
	return buflen;
}

