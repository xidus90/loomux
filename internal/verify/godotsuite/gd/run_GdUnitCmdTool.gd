extends "res://addons/gdUnit4/bin/GdUnitCmdTool.gd"
## loomux's gdUnit4 entry point: GdUnitCmdTool plus, when a shard names its
## directory, a coverage hook joined to the session in memory.
##
## The file name must contain "GdUnitCmdTool.gd": gdUnit4's argument parser
## skips every argument until it meets one that does.

const SHARD_ENV = "LOOMUX_COVERAGE_SHARD_DIR"
const IGNORE_SETTING = "nano_coverage/general/ignore_paths"
const GLUE_PATTERN = "**/.godot/loomux/**"


func _initialize() -> void:
	_ignore_glue()
	if OS.get_environment(SHARD_ENV) != "":
		# Looked up by name at run time: a static reference to a class this
		# gdUnit4 lacks would stop this script compiling, and the version
		# message below would never be printed.
		var service = _hook_service()
		if service == null or not ("enigne_hooks" in service):
			printerr(
				"loomux: this gdUnit4 (%s) has no session hook list to join" % _gdunit_version()
			)
			quit(70)
			return
		var hook = load("res://.godot/loomux/hook.gd").new()
		hook.set_meta("enabled", true)
		# Appended and not registered: register() saves the list into
		# project.godot, and assigning the array does not type-check.
		service.enigne_hooks.append(hook)
	super._initialize()


func _hook_service() -> Object:
	for entry in ProjectSettings.get_global_class_list():
		if entry["class"] == "GdUnitTestSessionHookService":
			return load(entry["path"]).instance()
	return null


## Nano instruments every script it can see, the hook included, and an
## instrumented hook aborts the engine. The setting is changed in memory
## only; get_setting returns Nano's own default when the key is absent, so
## appending keeps whatever the project ignores already.
func _ignore_glue() -> void:
	var ignored: PackedStringArray = ProjectSettings.get_setting(
		IGNORE_SETTING, PackedStringArray()
	)
	ignored.append(GLUE_PATTERN)
	ProjectSettings.set_setting(IGNORE_SETTING, ignored)


func _gdunit_version() -> String:
	var cfg := ConfigFile.new()
	cfg.load("res://addons/gdUnit4/plugin.cfg")
	return cfg.get_value("plugin", "version", "?")
