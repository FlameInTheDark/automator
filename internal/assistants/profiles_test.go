package assistants

import (
	"strings"
	"testing"
)

func TestDefaultPipelineEditorProfileIncludesLuaGuide(t *testing.T) {
	t.Parallel()

	profile := DefaultProfile(ScopePipelineEditor)

	if !containsString(profile.EnabledModules, "lua_scripting_guide") {
		t.Fatalf("default enabled modules = %#v, want lua_scripting_guide to be enabled", profile.EnabledModules)
	}
}

func TestBuildPromptAppendixIncludesLuaAndTemplateParameterGuidance(t *testing.T) {
	t.Parallel()

	appendix := BuildPromptAppendix(Profile{
		SystemInstructions: "Base instructions.",
		EnabledModules: []string{
			"templating_guide",
			"lua_scripting_guide",
		},
	})

	for _, snippet := range []string{
		"Base instructions.",
		"Preferred assistant skills:",
		"templating-guide",
		"lua-scripting-guide",
	} {
		if !strings.Contains(appendix, snippet) {
			t.Fatalf("prompt appendix missing %q:\n%s", snippet, appendix)
		}
	}

	for _, unwanted := range []string{
		"The current node input is exposed as a global named input.",
		`Returning a primitive becomes {"result": <value>}`,
	} {
		if strings.Contains(appendix, unwanted) {
			t.Fatalf("prompt appendix should not inline skill contents %q:\n%s", unwanted, appendix)
		}
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
