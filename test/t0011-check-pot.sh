#!/bin/sh

test_description="test git-po-helper check-po on .pot (CamelCase config check)"

. ./lib/test-lib.sh

HELPER="po-helper"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir
'

test_expect_success "prepare pot file" '
	git -C workdir checkout po-2.31.1 &&
	test -f workdir/po/git.pot &&
	sed -e "s|\(Project-Id-Version:\) PACKAGE VERSION|\1Git|" \
		workdir/po/git.pot >workdir/po/git.pot.tmp &&
	mv workdir/po/git.pot.tmp workdir/po/git.pot
'


test_expect_success "check-po on git.pot (Git project, CamelCase check)" '
	test_must_fail git -C workdir po-helper check-po po/git.pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	cat >expect <<-EOF &&
	level=error msg="config variable ${SQ}i18n.commitEncoding${SQ} in manpage does not match string in pot file:"
	level=error msg=" >> \"variable i18n.commitencoding to the encoding your project uses.\\\\n\""
	level=error msg="config variable ${SQ}submodule.fetchJobs${SQ} in manpage does not match string in pot file:"
	level=error msg=" >> msgid \"negative values not allowed for submodule.fetchjobs\""
	level=error msg="config variable ${SQ}gc.logExpiry${SQ} in manpage does not match string in pot file:"
	level=error msg=" >> msgid \"failed to parse gc.logexpiry value %s\""
	level=error msg="config variable ${SQ}pack.indexVersion${SQ} in manpage does not match string in pot file:"
	level=error msg=" >> msgid \"bad pack.indexversion=%<PRIu32>\""
	level=error msg="config variable ${SQ}pack.writeBitmaps${SQ} in manpage does not match string in pot file:"
	level=error msg=" >> \"--no-write-bitmap-index or disable the pack.writebitmaps configuration.\""
	level=error msg="config variable ${SQ}http.postBuffer${SQ} in manpage does not match string in pot file:"
	level=error msg=" >> msgid \"negative value for http.postbuffer; defaulting to %d\""
	level=error msg="6 mismatched config variables"
	ERROR: check-po command failed
	EOF

	test_cmp expect actual
'

test_done
