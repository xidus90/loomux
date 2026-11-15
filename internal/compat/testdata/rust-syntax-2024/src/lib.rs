pub struct Square(pub u32);

impl Square {
    pub fn area(&self) -> u32 {
        self.0 * self.0
    }
}

pub fn first_even(values: &[u32]) -> Option<u32> {
    if let Some(v) = values.first()
        && v % 2 == 0
    {
        Some(*v)
    } else {
        None
    }
}

pub fn r#gen(seed: u32) -> u32 {
    seed + 1
}

pub fn chars<'a>(text: &'a str) -> impl Iterator<Item = char> + use<'a> {
    text.chars()
}

pub async fn load(square: Square) -> u32 {
    let area = async |extra: u32| square.area() + extra;
    area(1).await
}

#[unsafe(no_mangle)]
pub extern "C" fn exported(value: i32) -> i32 {
    value + 1
}

unsafe extern "C" {
    pub safe fn abs(input: i32) -> i32;
    pub unsafe fn strlen(text: *const std::ffi::c_char) -> usize;
}

pub fn magnitude(value: i32) -> i32 {
    abs(value)
}
