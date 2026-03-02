#!/bin/sh

test_description="test git-po-helper check-pot"

. ./lib/test-lib.sh

HELPER="po-helper"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir
'

cat >expect <<-\EOF
level=error msg="config variable 'i18n.commitEncoding' in manpage does not match string in pot file:"
level=error msg=" >> \"variable i18n.commitencoding to the encoding your project uses.\\n\""
level=error msg="config variable 'submodule.fetchJobs' in manpage does not match string in pot file:"
level=error msg=" >> msgid \"negative values not allowed for submodule.fetchjobs\""
level=error msg="config variable 'gc.logExpiry' in manpage does not match string in pot file:"
level=error msg=" >> msgid \"failed to parse gc.logexpiry value %s\""
level=error msg="config variable 'pack.indexVersion' in manpage does not match string in pot file:"
level=error msg=" >> msgid \"bad pack.indexversion=%<PRIu32>\""
level=error msg="config variable 'pack.writeBitmaps' in manpage does not match string in pot file:"
level=error msg=" >> \"--no-write-bitmap-index or disable the pack.writebitmaps configuration.\""
level=error msg="config variable 'http.postBuffer' in manpage does not match string in pot file:"
level=error msg=" >> msgid \"negative value for http.postbuffer; defaulting to %d\""
ERROR: 6 mismatched config variables
EOF

test_expect_success "check-pot on git 2.31.1" '
	git -C workdir checkout po-2.31.1 &&
	test -f workdir/po/git.pot &&
	test_must_fail git -C workdir po-helper check-pot >out 2>&1 &&
	make_user_friendly_and_stable_output <out >actual &&

	test_cmp expect actual
'

test_done
