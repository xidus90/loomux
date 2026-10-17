extends SceneTree
## Rewrites the project's GDScript sources in place before the suites; loomux
## puts them back itself, from the manifest the addon leaves beside the
## backups, because a killed run leaves no process to start an engine with.
##
## On disk and not in memory because the in-memory pass cannot reach the
## autoloads: Nano skips every autoload by name, as patching a script the engine
## has already started would wipe its state, and names disk instrumentation as
## the way to cover them.
##
## save_project_settings must be false: left on, the addon registers its runtime
## autoload and saves project.godot, which reorders it and drops its comments.

const IGNORE_SETTING = "nano_coverage/general/ignore_paths"
const GLUE_PATTERN = "**/.godot/loomux/**"


func _init() -> void:
	# Keeps the glue scripts out of the rewrite, as in the runner.
	var ignored: PackedStringArray = ProjectSettings.get_setting(IGNORE_SETTING, PackedStringArray())
	ignored.append(GLUE_PATTERN)
	ProjectSettings.set_setting(IGNORE_SETTING, ignored)

	var result := DiskInstrumenter.new().instrument_to_disk({"save_project_settings": false})
	print(JSON.stringify(result))
	# The exit code decides, not the printed status: the caller reads a process
	# result, not prose.
	quit(0 if result.get("status", "") == "ok" else 1)
