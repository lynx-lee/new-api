package common

import "regexp"

// ccSwitchContextSuffixRe matches cc-switch context-length suffixes like [1m], [128k]
// at the end of model names. Examples: kimi-k3[1m] -> kimi-k3, deepseek-v4-pro[1m] -> deepseek-v4-pro.
//   - k = thousand tokens (e.g. [32k], [128k])
//   - m = million tokens (e.g. [1m], [2m])
var ccSwitchContextSuffixRe = regexp.MustCompile(`(?i)\[\d+[km]\]$`)

// StripCCSwitchContextSuffix removes cc-switch context-length suffix from a model name.
func StripCCSwitchContextSuffix(model string) string {
	return ccSwitchContextSuffixRe.ReplaceAllString(model, "")
}
