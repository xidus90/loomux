extends SceneTree
## Rewrites the project's GDScript sources in place before the suites and puts
## them back afterwards; loomux brackets the whole fan-out with it.
##
## On disk and not in memory because the in-memory pass cannot reach the
## autoloads: Nano skips every autoload by name, as patching a script the engine
## has already started would wipe its state, and names disk instrumentation as
## the way to cover them.
##
## save_project_settings must be false: left on, the addon registers its runtime
## autoload and saves project.godot, which reorders it and drops its comments.

const INSTRUMENT := "instrument"
const RESTORE := "restore"
const IGNORE_SETTING = "nano_coverage/general/ignore_paths"
const GLUE_PATTERN = "**/.godot/loomux/**"


func _init() -> void:
	var mode := INSTRUMENT
	for argument in OS.get_cmdline_user_args():
		mode = argument
	if mode != INSTRUMENT and mode != RESTORE:
		printerr("loomux: unknown mode '%s', expected instrument or restore" % mode)
		quit(2)
		return
	# Keeps the glue scripts out of the rewrite, as in the runner.
	var ignored: PackedStringArray = ProjectSettings.get_setting(IGNORE_SETTING, PackedStringArray())
	ignored.append(GLUE_PATTERN)
	ProjectSettings.set_setting(IGNORE_SETTING, ignored)

	var instrumenter := DiskInstrumenter.new()
	var result: Dictionary
	if mode == RESTORE:
		# Nothing to put back is a success: the bracket restores
		# unconditionally, and a run that never instrumented must not report a
		# broken restore.
		if not instrumenter.is_instrumented():
			print(JSON.stringify({"status": "ok", "restored_count": 0}))
			quit(0)
			return
		result = instrumenter.restore_from_disk({})
	else:
		result = instrumenter.instrument_to_disk({"save_project_settings": false})
	print(JSON.stringify(result))
	# The exit code decides, not the printed status: the caller reads a process
	# result, not prose.
	quit(0 if result.get("status", "") == "ok" else 1)
