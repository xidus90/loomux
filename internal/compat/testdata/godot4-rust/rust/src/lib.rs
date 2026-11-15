use godot::prelude::*;

mod player;

struct DemoExtension;

#[gdextension]
unsafe impl ExtensionLibrary for DemoExtension {}
