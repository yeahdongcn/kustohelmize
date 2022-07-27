package template

const (
	defaultIndent                 = "  "
	singleLineKeyFormat           = "%s: "
	globalSingleLineValueFormat   = "{{ include \"%s\" . }}"
	perFileSingleLineValueFormat  = "{{ .Values.%s.%s }}"
	perChartSingleLineValueFormat = "{{ .Values.%s }}"

	multilineKeyFormat           = "%s:"
	globalMultilineValueFormat   = "{{- include \"%s\" . | nindent %d }}"
	perFileMultilineValueFormat  = "{{ .Values.%s.%s | nindent %d }}"
	perChartMultilineValueFormat = "{{ .Values.%s | nindent %d }}"
	globalWithMixedFormat        = `{{- with %s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`
	perFileWithMixedFormat = `{{- with .Values.%s.%s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`
	perChartWithMixedFormat = `{{- with .Values.%s }}
%s:
  {{- toYaml . | nindent %d }}
{{- end }}`

	slicePrefixFirstLineFormat = "\n- "
	slicePrefixOtherLineFormat = "- "
)
