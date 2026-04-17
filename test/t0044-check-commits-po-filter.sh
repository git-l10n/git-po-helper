#!/bin/sh

test_description="check-commits applies PO filter check using repo path"

. ./lib/test-lib.sh

HELPER="po-helper --no-special-gettext-versions"
POT_NO="--pot-file=no"

test_expect_success "setup" '
	git clone "$PO_HELPER_TEST_REPOSITORY" workdir
'

# Use core.attributesfile so *.po filter applies without committing a root
# .gitattributes file (which would fail the "only po/" l10n change rule).
test_expect_success "check-commits fails when committed po does not match filter clean" '
	(
		cd workdir &&
		printf "%s\n" "po/*.po filter=gettext-no-location" >extra-attrs &&
		git config core.attributesfile "$(pwd)/extra-attrs" &&
		cat >po/sw.po <<-\EOF &&
	msgid ""
	msgstr ""
	"Project-Id-Version: Git\n"
	"Content-Type: text/plain; charset=UTF-8\n"

	#: tiny.c
	msgid "probe-filter-msg"
	msgstr "probe-filter-msg"
	EOF
		git add po/sw.po &&
		cat >.git/commit-message <<-\EOF &&
		l10n: test: add sw.po with locations under filter

		Signed-off-by: Author <author@example.com>
		EOF
		test_tick &&
		git commit -F .git/commit-message
	) &&
	test_must_fail git -C workdir $HELPER \
		check-commits --report-file-locations=error $POT_NO HEAD~..HEAD >out 2>&1 &&
	test_grep "PO filter (.gitattributes)" out
'

test_expect_success "check-commits passes filter mismatch when --no-check-filter" '
	git -C workdir $HELPER \
		check-commits --no-check-filter --report-file-locations=error \
		$POT_NO HEAD~..HEAD >out 2>&1 &&
	test_must_fail grep -q "PO filter (.gitattributes)" out
'

test_done
