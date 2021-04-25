package data

func init() {
	langMap = make(map[string]string)
        {{- range $key, $value := . }}
        langMap["{{ $key }}"] = "{{ $value }}"
        {{- end }}
}