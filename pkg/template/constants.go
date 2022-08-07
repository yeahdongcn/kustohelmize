package template

const (
	leftDelimiter  = "{{ "
	rightDelimiter = " }}"

	leftDelimiterTrimSpaceTrailing = "{{- "
	rightDelimiterTrimSpaceLeading = " -}}"
)

const (
	singleValueFormat   = leftDelimiter + "%s" + rightDelimiter
	singleIncludeFormat = leftDelimiter + "include \"%s\" ." + rightDelimiter

	newlineValueFormat   = leftDelimiterTrimSpaceTrailing + "%s | nindent %d" + rightDelimiter
	newlineIncludeFormat = leftDelimiterTrimSpaceTrailing + "include \"%s\" . | nindent %d" + rightDelimiter

	withFormat = leftDelimiterTrimSpaceTrailing + "with %s" + rightDelimiter + "\n%s:\n" +
		"  " + leftDelimiterTrimSpaceTrailing + "toYaml . | nindent %d" + rightDelimiter + "\n" +
		leftDelimiterTrimSpaceTrailing + "end" + rightDelimiter
)

const (
	defaultIndent = "  "

	singleLineKeyFormat = "%s: "

	newlineKeyFormat = "%s:"

	slicePrefixFirstLineFormat = "\n- "
	slicePrefixOtherLineFormat = "- "
)
