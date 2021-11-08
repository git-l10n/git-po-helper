# Format the output of git-push, git-show-ref and other commands to make a
# user-friendly and stable text.  We can easily prepare the expect text
# without having to worry about future changes of the commit ID and spaces
# of the output.  Single quotes are replaced with double quotes, because
# it is boring to prepare unquoted single quotes in expect text.  We also
# remove some locale error messages. The emitted human-readable errors are
# redundant to the more machine-readable output the tests already assert.
make_user_friendly_and_stable_output () {
	sed \
		-e "/Please check your system clock/d" \
		-e "s/  *\$//" \
		-e "s/  */ /g" \
		-e "s/	/    /g" \
		-e "s/\\\\t/    /g" \
		-e "s#/[a-zA-Z/]*/msgfmt#msgfmt#g" \
		-e "s/po@[0-9a-f][0-9a-f]*\]/po@rev]/g" \
		-e "s/(use \".*\" for backward compatible)/(use \"gettext 0.14\" for backward compatible)/" \
		-e "s/$ZERO_OID/<ZERO-OID>/g" \
		-e "s/commit [0-9a-f][0-9a-f]*\([: ]\)/commit <OID>\1/g" \
		-e "s/$OID_REGEX/<OID>/g" \
		-e "s/illegal byte sequence/<iconv failure message>.../" \
		-e "s/invalid or incomplete multibyte or wide character/<iconv failure message>.../" |
	perl -pe "s/\e\[[0-9;]*m//g"

}
