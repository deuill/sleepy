# Copyright 2012 - 2014 Alex Palaistras. All rights reserved.
# Use of this source code is governed by the MIT License, the
# full text of which can be found in the LICENSE file.
#
# Makefile for Sleepy server component. Run 'make' to build the sleepy binary,
# 'make install' to install the binary part and 'make install-data' to install
# the common files (configuration and database).
# 
# User-defined build options.
# 
COMPILER = gc
PROGRAM = sleepyd
VERSION = 0.5.0

# No editing from here on!
# 
.PHONY: $(PROGRAM)
all: $(PROGRAM)

$(PROGRAM):
	@echo -e "\033[1mBuilding '$(PROGRAM)'...\033[0m"

	@mkdir -p .tmp
	@go build -compiler $(COMPILER) -o .tmp/$(PROGRAM) ./core

install:
	@echo -e "\033[1mInstalling binaries...\033[0m"

	@install -s -Dm 755 .tmp/$(PROGRAM) $(DESTDIR)/usr/bin/$(PROGRAM)

install-data:
	@echo -e "\033[1mInstalling data...\033[0m"

	@install -d $(DESTDIR)/etc/sleepy/modules.d $(DESTDIR)/etc/sleepy/disabled.d
	@install -d $(DESTDIR)/var/lib/sleepy

	@install -m 644 data/conf/*.conf $(DESTDIR)/etc/sleepy
	@install -m 644 data/conf/modules.d/*.conf $(DESTDIR)/etc/sleepy/modules.d
	@install -m 644 data/sleepy.db $(DESTDIR)/var/lib/sleepy

package:
	@echo -e "\033[1mBuilding package...\033[0m"

	@mkdir -p .tmp/package
	@make DESTDIR=.tmp/package install install-data
	@tar -cJf sleepy-$(VERSION).tar.xz -C .tmp/package .

uninstall:
	@echo -e "\033[1mUninstalling...\033[0m"

	@rm -Rf $(DESTDIR)/etc/sleepy
	@rm -f $(DESTDIR)/usr/bin/$(PROGRAM)

clean:
	@echo -e "\033[1mCleaning...\033[0m"

	@go clean
	@rm -Rf .tmp
