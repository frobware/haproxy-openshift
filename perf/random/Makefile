CFLAGS ?= -ggdb3 -O0 -Og -Wall -Wextra -Werror -fPIC -D_GNU_SOURCE -fno-omit-frame-pointer

all: libmyrandom.so

libmyrandom.so: getrandom-interposer.o
	$(CC) $(CFLAGS) -W -shared -Wl,-soname,$@ -Wl,--no-as-needed -o $@ -ldl $<

getrandom-interposer.o: Makefile

clean:
	$(RM) -f *.o

.PHONY: all clean
