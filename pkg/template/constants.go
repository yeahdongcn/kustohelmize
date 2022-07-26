package template

const (
	defaultIndent                = "  "
	singleLineKeyFormat          = "%s: "
	globalSingleLineValueFormat  = "{{ include \"%s\" . }}"
	perFileSingleLineValueFormat = "{{ .Values.%s.%s }}"
	multilineKeyFormat           = "%s:"
	globalMultilineValueFormat   = "{{- include \"%s\" . | nindent %d }}"
	perFileMultilineValueFormat  = "{{ .Values.%s.%s | nindent %d }}"
	withMixedFormat              = `{{- with %s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`
	slicePrefixFirstLineFormat = "\n- "
	slicePrefixOtherLineFormat = "- "
)
