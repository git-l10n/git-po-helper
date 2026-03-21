package data

func init() {
	scriptMap = make(map[string]string)
        {{- range $key, $value := . }}
        scriptMap["{{ $key }}"] = "{{ goescape $value }}"
        {{- end }}
}
