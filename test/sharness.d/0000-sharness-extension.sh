#!/bin/sh

: "${PERL_PATH:=/usr/bin/perl}"
export PERL_PATH

exec 7>&2

BUG () {
	error >&7 "bug in the test script: $*"
}

TEST_AUTHOR_LOCALNAME=author
TEST_AUTHOR_DOMAIN=example.com
GIT_AUTHOR_EMAIL=${TEST_AUTHOR_LOCALNAME}@${TEST_AUTHOR_DOMAIN}
GIT_AUTHOR_NAME='A U Thor'
GIT_AUTHOR_DATE='1112354055 +0200'
TEST_COMMITTER_LOCALNAME=committer
TEST_COMMITTER_DOMAIN=example.com
GIT_COMMITTER_EMAIL=${TEST_COMMITTER_LOCALNAME}@${TEST_COMMITTER_DOMAIN}
GIT_COMMITTER_NAME='C O Mitter'
GIT_COMMITTER_DATE='1112354055 +0200'
GIT_MERGE_VERBOSITY=5
GIT_MERGE_AUTOEDIT=no
export GIT_MERGE_VERBOSITY GIT_MERGE_AUTOEDIT
export GIT_AUTHOR_EMAIL GIT_AUTHOR_NAME
export GIT_COMMITTER_EMAIL GIT_COMMITTER_NAME
export GIT_COMMITTER_DATE GIT_AUTHOR_DATE
export EDITOR

GIT_DEFAULT_HASH="${GIT_TEST_DEFAULT_HASH:-sha1}"
export GIT_DEFAULT_HASH

# Tests using GIT_TRACE typically don't want <timestamp> <file>:<line> output
GIT_TRACE_BARE=1
export GIT_TRACE_BARE

# Use fixed commit time
test_tick () {
	if test -z "${test_tick+set}"
	then
		test_tick=1112911993
	else
		test_tick=$(($test_tick + 60))
	fi
	GIT_COMMITTER_DATE="$test_tick -0700"
	GIT_AUTHOR_DATE="$test_tick -0700"
	export GIT_COMMITTER_DATE GIT_AUTHOR_DATE
}

# Set the hash algorithm in use to $1.  Only useful when testing the testsuite.
test_set_hash () {
	test_hash_algo="$1"
}

# Detect the hash algorithm in use.
test_detect_hash () {
	test_hash_algo="${GIT_TEST_DEFAULT_HASH:-sha1}"
}


# Load common hash metadata and common placeholder object IDs for use with
# test_oid.
test_oid_init () {
	test -n "$test_hash_algo" || test_detect_hash &&
	test_oid_cache <"$SHARNESS_TEST_DIRECTORY/lib/oid-info/hash-info" &&
	test_oid_cache <"$SHARNESS_TEST_DIRECTORY/lib/oid-info/oid"
}

# Load key-value pairs from stdin suitable for use with test_oid.  Blank lines
# and lines starting with "#" are ignored.  Keys must be shell identifier
# characters.
#
# Examples:
# rawsz sha1:20
# rawsz sha256:32
test_oid_cache () {
	local tag rest k v &&

	{ test -n "$test_hash_algo" || test_detect_hash; } &&
	while read tag rest
	do
		case $tag in
		\#*)
			continue;;
		?*)
			# non-empty
			;;
		*)
			# blank line
			continue;;
		esac &&

		k="${rest%:*}" &&
		v="${rest#*:}" &&

		if ! expr "$k" : '[a-z0-9][a-z0-9]*$' >/dev/null
		then
			BUG 'bad hash algorithm'
		fi &&
		eval "test_oid_${k}_$tag=\"\$v\""
	done
}

# Look up a per-hash value based on a key ($1).  The value must have been loaded
# by test_oid_init or test_oid_cache.
test_oid () {
	local algo="${test_hash_algo}" &&

	case "$1" in
	--hash=*)
		algo="${1#--hash=}" &&
		shift;;
	*)
		;;
	esac &&

	local var="test_oid_${algo}_$1" &&

	# If the variable is unset, we must be missing an entry for this
	# key-hash pair, so exit with an error.
	if eval "test -z \"\${$var+set}\""
	then
		BUG "undefined key '$1'"
	fi &&
	eval "printf '%s' \"\${$var}\""
}

# Insert a slash into an object ID so it can be used to reference a location
# under ".git/objects".  For example, "deadbeef..." becomes "de/adbeef..".
test_oid_to_path () {
	local basename=${1#??}
	echo "${1%$basename}/$basename"
}

# Convenience
# A regexp to match 5, 35 and 40 hexdigits
_x05='[0-9a-f][0-9a-f][0-9a-f][0-9a-f][0-9a-f]'
_x35="$_x05$_x05$_x05$_x05$_x05$_x05$_x05"
_x40="$_x35$_x05"

test_oid_init

ZERO_OID=$(test_oid zero)
OID_REGEX=$(echo $ZERO_OID | sed -e 's/0/[0-9a-f]/g')
OIDPATH_REGEX=$(test_oid_to_path $ZERO_OID | sed -e 's/0/[0-9a-f]/g')
EMPTY_TREE=$(test_oid empty_tree)
EMPTY_BLOB=$(test_oid empty_blob)
_z40=$ZERO_OID

# UTF-8 ZERO WIDTH NON-JOINER, which HFS+ ignores
# when case-folding filenames
u200c=$(printf '\342\200\214')

# Line feed
LF='
'

# Single quote
SQ=\'

export _x05 _x35 _x40 _z40 LF u200c EMPTY_TREE EMPTY_BLOB ZERO_OID OID_REGEX

# Run test_tick to initial author/committer name and time
test_tick
