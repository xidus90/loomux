package linecov

// GodotCfg reads one section of project.godot for callers outside the gate.
func GodotCfg(content []byte, section string) (Values, bool, error) {
	return readGodotCfg(content, Where{Section: section})
}

// NanoMatcher reads a pattern the way Nano Coverage's script scanner does.
func NanoMatcher(pattern string) (Matcher, error) { return nanoGlob(pattern, "") }
