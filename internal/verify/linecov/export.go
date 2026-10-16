package linecov

// GodotCfg reads one section of project.godot for callers outside the gate.
// A key set to an empty string is left out, as in the gate.
func GodotCfg(content []byte, section string) (Values, bool, error) {
	return readGodotCfg(content, Where{Section: section})
}

// GodotCfgKeepEmpty is GodotCfg for a caller that must tell a key set to an
// empty string (one empty value) from an absent key.
func GodotCfgKeepEmpty(content []byte, section string) (Values, bool, error) {
	return readGodotCfg(content, Where{Section: section, KeepEmpty: true})
}

// NanoMatcher reads a pattern the way Nano Coverage's script scanner does.
func NanoMatcher(pattern string) (Matcher, error) { return nanoGlob(pattern, "") }
