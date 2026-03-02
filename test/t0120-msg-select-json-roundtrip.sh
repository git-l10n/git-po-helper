#!/bin/sh
#
# Test PO -> gettext JSON -> PO round-trip with msgcat normalization.
# Uses a sample from zh_CN.po (test/fixtures/zh_CN_example.po).
# After round-trip, both PO files are formatted with msgcat and compared.
#

test_description="msg-select gettext JSON round-trip: PO -> JSON -> PO, compare with msgcat"

. ./lib/test-lib.sh

# Need msgcat (gettext) to normalize PO for comparison
if ! command -v msgcat >/dev/null 2>&1; then
	skip_all="msgcat (gettext) not found, skip gettext JSON round-trip test"
	test_done
fi

# Use the built binary (same as other tests that run po-helper via git)
HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"
FIXTURE="$TEST_DIRECTORY/fixtures/zh_CN_example.po"

test_expect_success "setup: fixture with entries exists" '
	test -f "$FIXTURE" &&
	head -1 "$FIXTURE" | grep -q "Chinese translations"
'

test_expect_success "PO -> JSON: msg-select --range 1- --json" '
	$HELPER msg-select --range "1-" --json "$FIXTURE" >sample.json &&
	test -s sample.json &&
	test_copy_bytes 1 <sample.json | grep -q "{"
'

test_expect_success "JSON -> PO: msg-select --range 1- (input is JSON)" '
	$HELPER msg-select --range "1-" -o roundtrip.po sample.json &&
	test -s roundtrip.po &&
	head -1 roundtrip.po | grep -q "^#"
'

test_expect_success "format both PO files with msgcat" '
	msgcat -o sample_fmt.po "$FIXTURE" &&
	msgcat -o roundtrip_fmt.po roundtrip.po &&
	test -s sample_fmt.po &&
	test -s roundtrip_fmt.po
'

test_expect_success "formatted PO files are identical after round-trip" '
	test_cmp sample_fmt.po roundtrip_fmt.po
'

test_done
