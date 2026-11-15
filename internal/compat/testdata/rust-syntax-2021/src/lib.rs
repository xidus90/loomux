use std::fmt::Display;

pub trait Shape {
    type Unit: Display;

    fn area(&self) -> u32;

    fn describe(&self) -> String {
        format!("{} units", self.area())
    }
}

pub struct Square<const N: u32>;

impl<const N: u32> Shape for Square<N> {
    type Unit = u32;

    fn area(&self) -> u32 {
        N * N
    }
}

macro_rules! twice {
    ($x:expr) => {
        $x * 2
    };
}

pub const fn doubled(value: u32) -> u32 {
    twice!(value)
}

pub fn parse(text: &str) -> Result<u32, std::num::ParseIntError> {
    let Some(first) = text.split(',').next() else {
        return Ok(0);
    };
    let value: u32 = first.trim().parse()?;
    Ok(doubled(value))
}

pub fn total(shapes: &[&dyn Shape<Unit = u32>]) -> u32 {
    shapes.iter().map(|s| s.area()).sum()
}

pub async fn load(shape: impl Shape<Unit = u32>) -> u32 {
    let area = async move { shape.area() };
    area.await
}
