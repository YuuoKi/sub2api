package service

// PrepareSimulationCapturedPrompt redacts and UTF-8-bounds a simulation prompt
// using the same hard cap and redaction_version as generation_content Collect.
func PrepareSimulationCapturedPrompt(prompt string) (promptRedacted string, promptBytes int, redactionVersion int) {
	promptBytes = len(prompt)
	promptInput := truncateValidUTF8([]byte(prompt), maxGenerationPromptMaxBytes)
	promptRedacted = truncateStringUTF8(redactGenerationPrompt(promptInput), maxGenerationPromptMaxBytes)
	return promptRedacted, promptBytes, generationRedactionVersion
}
