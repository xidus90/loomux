pub fn one() -> i32 { 1 }
// only reachable from the CLI binary
#[cfg_attr(coverage_nightly, coverage(off))]
pub fn two() -> i32 { 2 }

mod shim {
    #![coverage(off)]
    pub fn three() -> i32 { 3 }
}

// LCOV_EXCL_START: platform glue the tests cannot reach
pub fn four() -> i32 {
    4
}
// LCOV_EXCL_STOP

// LCOV_EXCL_START
pub fn five() -> i32 {
    5
}
// LCOV_EXCL_STOP

// LCOV_EXCL_START: never closed
pub fn six() -> i32 {
    6
}
