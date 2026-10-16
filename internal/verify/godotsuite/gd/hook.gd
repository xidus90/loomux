extends GdUnitTestSessionHook
## Nano Coverage session hook for one loomux shard: instruments the scripts
## before the tests and writes an LCOV report when the session ends, into the
## shard's own directory.
##
## It is a file of its own instead of a patch to the addon, which is vendored
## and would need the patch again with every update. Nothing is registered in
## project.godot; run_GdUnitCmdTool.gd attaches it.

const SHARD_ENV = "LOOMUX_COVERAGE_SHARD_DIR"
const DATA_STORE_SETTING = "nano_coverage/general/data_store_dir"
const REPORT_DIR_SETTING = "nano_coverage/general/report_dir"
const DEFAULT_DATA_STORE = "res://coverage-data"
const METADATA_NAME = "coverage.meta"

var _workspace := "loomux"


func _init() -> void:
	super("LoomuxCoverageShard", "Instrument scripts and write this shard's LCOV report.")


func startup(session: GdUnitTestSession) -> GdUnitResult:
	var dir := OS.get_environment(SHARD_ENV)
	# Read before the redirect below: the disk pass left its line metadata in
	# the store the project configures, not in the shard's.
	var configured: String = ProjectSettings.get_setting(DATA_STORE_SETTING, DEFAULT_DATA_STORE)
	# Redirected before instrumenting, so a run that dies half-way has already
	# left its fragments in the shard's directory and not in a shared one.
	var data := dir.path_join("data")
	var report := dir.path_join("report")
	ProjectSettings.set_setting(DATA_STORE_SETTING, data)
	ProjectSettings.set_setting(REPORT_DIR_SETTING, report)
	_workspace = "loomux-%s" % dir.get_file()

	var api := CoverageApi.new()
	api.clear_coverage_data({"workspace_id": _workspace})

	if DiskInstrumenter.new().is_instrumented():
		# Instrumenting again would double every counter. The metadata has to
		# be copied because clear_coverage_data has just emptied this shard's
		# store; without it the report fails with "Metadata file missing".
		var source := configured.path_join(METADATA_NAME)
		DirAccess.make_dir_recursive_absolute(data)
		var copied := DirAccess.copy_absolute(source, data.path_join(METADATA_NAME))
		if copied != OK:
			# A message and not an error: the missing LCOV is what the caller
			# reports, as a shard without metadata writes no report.
			session.send_message("NanoCoverage: cannot copy %s to %s" % [source, data])
			return GdUnitResult.success()
	else:
		ProjectBootstrapper.new().instrument_all_scripts()

	session.send_message("NanoCoverage: shard %s, workspace %s." % [dir, _workspace])
	return GdUnitResult.success()


func shutdown(session: GdUnitTestSession) -> GdUnitResult:
	if not Engine.has_singleton("NanoCoverage"):
		# A message and not a crash: nothing was measured, and the caller has
		# to see an absent report rather than an exception that reads like a
		# broken suite.
		session.send_message("NanoCoverage: singleton not found, no report written.")
		return GdUnitResult.success()

	var nc := Engine.get_singleton("NanoCoverage")
	nc.save_session(_workspace)
	nc.reset()

	# Untyped on purpose: a typed assignment would turn a failed generation
	# that returns null into a script error after reset() has discarded the
	# hits, leaving neither a report nor a message.
	var result = CoverageApi.new().generate_coverage_report({"workspace_id": _workspace})
	if result is Dictionary and result.get("status", "") == "ok":
		session.send_message("NanoCoverage: LCOV report in %s" % result.get("report_path", "?"))
	else:
		session.send_message("NanoCoverage: report generation failed: %s" % [result])
	return GdUnitResult.success()
