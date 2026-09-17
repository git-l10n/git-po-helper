#!/bin/sh
#
# VERSION-GEN fallback: version file, then git describe, then CHANGELOG.md.
#

test_description="VERSION-GEN version sources"

. ./lib/test-lib.sh

ROOT="$(cd "$TEST_DIRECTORY/.." && pwd)"

test_expect_success "VERSION-GEN reads first CHANGELOG ## X.Y.Z without git" '
	mkdir nogit &&
	cp "$ROOT/VERSION-GEN" nogit/VERSION-GEN &&
	cat >nogit/CHANGELOG.md <<-EOF &&
	# Changelog

	## 1.2.3 (2099-01-01)

	* note

	## 0.1.0 (2098-01-01)
	EOF
	(
		cd nogit &&
		rm -f VERSION-FILE version &&
		/bin/sh ./VERSION-GEN &&
		grep "^VERSION = 1.2.3$" VERSION-FILE
	)
'

test_expect_success "VERSION-GEN prefers version file over CHANGELOG" '
	mkdir withver &&
	cp "$ROOT/VERSION-GEN" withver/VERSION-GEN &&
	cat >withver/CHANGELOG.md <<-EOF &&
	## 9.9.9 (2099-01-01)
	EOF
	echo 3.4.5 >withver/version &&
	(
		cd withver &&
		rm -f VERSION-FILE &&
		/bin/sh ./VERSION-GEN &&
		grep "^VERSION = 3.4.5$" VERSION-FILE
	)
'

test_expect_success "VERSION-GEN rejects two-component CHANGELOG headings" '
	mkdir shortver &&
	cp "$ROOT/VERSION-GEN" shortver/VERSION-GEN &&
	cat >shortver/CHANGELOG.md <<-EOF &&
	## 1.0 (bad tag shape)

	## 2.3.4 (2099-01-01)
	EOF
	(
		cd shortver &&
		rm -f VERSION-FILE version &&
		/bin/sh ./VERSION-GEN &&
		grep "^VERSION = 2.3.4$" VERSION-FILE
	)
'

test_expect_success "VERSION-GEN defaults when no version sources" '
	mkdir empty &&
	cp "$ROOT/VERSION-GEN" empty/VERSION-GEN &&
	(
		cd empty &&
		rm -f VERSION-FILE version CHANGELOG.md &&
		/bin/sh ./VERSION-GEN &&
		grep "^VERSION = 0.0.0$" VERSION-FILE
	)
'

test_done
