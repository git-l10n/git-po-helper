package data

func init() {
	locationMap = make(map[string]string)
        {{- range $key, $value := . }}
        locationMap["{{ $key }}"] = "{{ $value }}"
        {{- end }}
}
