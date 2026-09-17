#!/bin/sh

test_description="test git-po-helper version"

. ./lib/test-lib.sh

HELPER="$TEST_TARGET_DIRECTORY/git-po-helper --no-special-gettext-versions"

test_expect_success "git-po-helper version output test" '
	$HELPER version >out &&
	grep "^git-po-helper version" out >expect &&
	test -s expect
'

test_expect_success "version has at least major.minor.patch (rejects v1.0-style tags)" '
	# Require >= 3 leading numeric dotted components from the built version.
	# Fails if the nearest describe tag is like v1.0 / v0.8 instead of v1.0.0.
	sed -n "s/^git-po-helper version //p" out >ver &&
	grep -E "^[0-9]+\.[0-9]+\.[0-9]+([.]|$)" ver >semver3 &&
	test_cmp semver3 ver
'

test_expect_success "check git-po-helper version format line" '
	grep "^git-po-helper version [0-9]\+\.[0-9]\+\.[0-9]\+" out >actual &&
	test_cmp expect actual
'

test_expect_success "version --ge 0 succeeds and still prints version" '
	$HELPER version --ge 0 >out 2>err &&
	test_must_be_empty err &&
	grep -q "^git-po-helper version " out &&
	! grep -q "^ERROR:" out
'

test_expect_success "version --lt 0 fails: version+ERROR on stdout, empty stderr" '
	test_must_fail $HELPER version --lt 0 >out 2>err &&
	test_must_be_empty err &&
	grep -q "^git-po-helper version " out &&
	grep -q "^ERROR: version .* does not satisfy --lt 0$" out
'

test_expect_success "version --eq major matches major-only precision" '
	# Use the running major (not a hardcoded 0) so 1.0.0 still passes.
	major=$(sed -n "s/^\([0-9][0-9]*\)\..*/\1/p" ver) &&
	test -n "$major" &&
	$HELPER version --eq "$major" >out 2>err &&
	test_must_be_empty err &&
	grep -q "^git-po-helper version " out
'

test_expect_success "version --eq patch matches describe builds" '
	# X.Y.Z.N.g… must satisfy --eq X.Y.Z (distance ignored).
	ver=$($HELPER version | sed -n "s/^git-po-helper version //p") &&
	patch=$(echo "$ver" | sed -n "s/^\([0-9]*\.[0-9]*\.[0-9]*\).*/\1/p") &&
	test -n "$patch" &&
	$HELPER version --eq "$patch" >out 2>err &&
	test_must_be_empty err
'

test_expect_success "version --eq wrong major fails" '
	major=$(sed -n "s/^\([0-9][0-9]*\)\..*/\1/p" ver) &&
	test -n "$major" &&
	wrong=$((major + 999)) &&
	test_must_fail $HELPER version --eq "$wrong" >out 2>err &&
	test_must_be_empty err &&
	grep -q "^ERROR: version .* does not satisfy --eq ${wrong}$" out
'

test_expect_success "version --ge and --lt are mutually exclusive" '
	test_must_fail $HELPER version --ge 0 --lt 1 2>err &&
	grep -q "mutually exclusive" err
'

test_expect_success "version --ge with invalid constraint" '
	test_must_fail $HELPER version --ge 0.8.x >out 2>err &&
	grep -q "^git-po-helper version " out &&
	grep -q "ERROR:" err
'

test_done
