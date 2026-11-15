use godot::prelude::*;

#[derive(GodotClass)]
#[class(init, base=Node2D)]
pub struct Player {
    #[init(val = 2.0)]
    speed: f64,
    base: Base<Node2D>,
}

#[godot_api]
impl Player {
    #[signal]
    fn hit();

    #[func]
    fn get_speed(&self) -> f64 {
        self.speed
    }

    #[func(rename = do_move)]
    fn move_by(&mut self, delta: f64) -> f64 {
        advance(self.speed, delta)
    }
}

fn advance(speed: f64, delta: f64) -> f64 {
    speed + delta
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn advance_adds_the_delta() {
        assert_eq!(advance(2.0, 1.5), 3.5);
    }
}
