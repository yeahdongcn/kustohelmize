package template

const (
	defaultIndent = "  "

	singleLineKeyFormat         = "%s: "
	globalSingleLineValueFormat = "{{ include \"%s\" . }}"
	fileSingleLineValueFormat   = "{{ .Values.%s.%s }}"
	sharedSingleLineValueFormat = "{{ .Values.%s }}"

	multilineKeyFormat         = "%s:"
	globalMultilineValueFormat = "{{- include \"%s\" . | nindent %d }}"
	fileMultilineValueFormat   = "{{ .Values.%s.%s | nindent %d }}"
	sharedMultilineValueFormat = "{{ .Values.%s | nindent %d }}"

	globalWithMixedFormat = `{{- with %s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`
	fileWithMixedFormat = `{{- with .Values.%s.%s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`
	sharedWithMixedFormat = `{{- with .Values.%s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`

	slicePrefixFirstLineFormat = "\n- "
	slicePrefixOtherLineFormat = "- "
)
