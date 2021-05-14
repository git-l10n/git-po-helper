if ! type xgettext
then
	if type apt-get
	then
		apt-get update &&
		apt-get install --assume-yes gettext
	elif type yum
	then
		yum makecache &&
		yum install -y gettext
	elif type brew
	then
		brew install gettext
	fi
fi
