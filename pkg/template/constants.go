package template

const (
	defaultIndent = "  "
)

const (
	leftDelimiter  = "{{ "
	rightDelimiter = " }}"

	leftDelimiterTrimSpaceTrailing = "{{- "
	rightDelimiterTrimSpaceLeading = " -}}"
)

const (
	slicePrefixFirstLineFormat = "\n- "
	slicePrefixOtherLineFormat = "- "
)

const (
	singleLineKeyFormat   = "%s: "
	singleValueFormat     = leftDelimiter + "%s" + rightDelimiter
	singleYAMLValueFormat = leftDelimiter + "toYaml %s" + rightDelimiter
	singleIncludeFormat   = leftDelimiter + "include \"%s\" ." + rightDelimiter

	newlineKeyFormat       = "%s:"
	newlineValueFormat     = leftDelimiterTrimSpaceTrailing + "%s | nindent %d" + rightDelimiter
	newlineYAMLValueFormat = leftDelimiterTrimSpaceTrailing + "toYaml %s | nindent %d" + rightDelimiter
	newlineIncludeFormat   = leftDelimiterTrimSpaceTrailing + "include \"%s\" . | nindent %d" + rightDelimiter

	withFormat = leftDelimiterTrimSpaceTrailing + "with %s" + rightDelimiter + "\n%s:\n" +
		"  " + leftDelimiterTrimSpaceTrailing + "toYaml . | nindent %d" + rightDelimiter + "\n" +
		leftDelimiterTrimSpaceTrailing + "end" + rightDelimiter

	ifFormat = leftDelimiterTrimSpaceTrailing + "if %s" + rightDelimiter + "\n%s:\n" +
		"  " + leftDelimiter + "%s | nindent %d" + rightDelimiter + "\n" +
		leftDelimiterTrimSpaceTrailing + "end" + rightDelimiter
	ifYAMLFormat = leftDelimiterTrimSpaceTrailing + "if %s" + rightDelimiter + "\n%s:\n" +
		"  " + leftDelimiter + "toYaml %s | nindent %d" + rightDelimiter + "\n" +
		leftDelimiterTrimSpaceTrailing + "end" + rightDelimiter
)
