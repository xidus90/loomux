pub struct Base;

impl Base {
    pub fn f(&self) -> i32 {
        1
    }
}

pub struct Child {
    base: Base,
}

impl Child {
    pub fn new() -> Self {
        Child { base: Base }
    }

    pub fn f(&self) -> i32 {
        self.base.f() + 1
    }
}

impl Default for Child {
    fn default() -> Self {
        Self::new()
    }
}

pub fn twice(value: i32) -> i32 {
    Child::new().f() * value
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn child_adds_one_to_the_base() {
        assert_eq!(Child::default().f(), 2);
    }

    #[test]
    fn twice_doubles_a_value() {
        assert_eq!(twice(2), 4);
    }
}
