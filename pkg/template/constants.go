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
	endDelimited          = leftDelimiterTrimSpaceTrailing + "end" + rightDelimiter
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
		endDelimited

	ifFormat = leftDelimiterTrimSpaceTrailing + "if %s" + rightDelimiter + "\n%s: " +
		leftDelimiter + "%s" + rightDelimiter + "\n" +
		endDelimited
	ifNotFormat = leftDelimiterTrimSpaceTrailing + "if not %s" + rightDelimiter + "\n%s: " +
		leftDelimiter + "%s" + rightDelimiter + "\n" +
		endDelimited
	ifYAMLFormat = leftDelimiterTrimSpaceTrailing + "if %s" + rightDelimiter + "\n%s: " +
		leftDelimiter + "toYaml %s | nindent %d" + rightDelimiter + "\n" +
		endDelimited
	ifNotYAMLFormat = leftDelimiterTrimSpaceTrailing + "if not %s" + rightDelimiter + "\n%s: " +
		leftDelimiter + "toYaml %s | nindent %d" + rightDelimiter + "\n" +
		endDelimited
	ifOriginFormat = leftDelimiterTrimSpaceTrailing + "if %s" + rightDelimiter + "\n%s: " +
		"%s\n" +
		endDelimited
	ifNotOriginFormat = leftDelimiterTrimSpaceTrailing + "if not %s" + rightDelimiter + "\n%s: " +
		"%s\n" +
		endDelimited

	fileIfFormat = leftDelimiterTrimSpaceTrailing + "if %s" + rightDelimiter + "\n"

	rangeFormat = "%s:\n" + leftDelimiterTrimSpaceTrailing + "range %s" + rightDelimiter + "\n" +
		"  - name: " + leftDelimiter + "." + rightDelimiter + "\n" +
		endDelimited

	appendWithFormat = leftDelimiterTrimSpaceTrailing + "with %s" + rightDelimiter + "\n" +
		leftDelimiterTrimSpaceTrailing + "toYaml . | nindent %d" + rightDelimiter + "\n" +
		endDelimited
)
