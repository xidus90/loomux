int one(void) { return 1; }
int two(void) { return 2; } /* LCOV_EXCL_LINE never called on this platform */
/* LCOV_EXCL_START legacy API */
int three(void) { return 3; }
/* LCOV_EXCL_STOP */
int four(void) { return 4; } /* NOCOV_LINE built without the codec */
/* NOCOV_START experimental */
int five(void) { return 5; }
/* NOCOV_STOP */
int six(void) { return 6; } /* LCOV_EXCL_LINE */
int seven(void) { return 7; } /* NOCOV_LINE */
