pub fn one() -> i32 { 1 }
// only reachable from the CLI binary
#[cfg_attr(coverage_nightly, coverage(off))]
pub fn two() -> i32 { 2 }

mod shim {
    #![coverage(off)]
    pub fn three() -> i32 { 3 }
}
