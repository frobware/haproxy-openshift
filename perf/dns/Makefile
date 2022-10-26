CFLAGS ?= -ggdb3 -O0 -Og -Wall -Wextra -Werror -Wno-unused-function -fPIC -D_GNU_SOURCE -fno-omit-frame-pointer

all: clean libmydns.so

libmydns.so: getaddrinfo-interposer.o
	$(CC) $(CFLAGS) -W -shared -Wl,-soname,$@ -Wl,--no-as-needed -o $@ -ldl $<

getaddrinfo-interposer.o: Makefile

clean:
	$(RM) -f *.o

.PHONY: all clean
