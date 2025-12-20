use std::collections::HashMap;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct RgbColor {
    pub r: u8,
    pub g: u8,
    pub b: u8,
}

pub fn default_colors() -> HashMap<&'static str, RgbColor> {
    HashMap::from([
        ("droid", RgbColor { r: 255, g: 140, b: 0 }),
        ("claude", RgbColor { r: 210, g: 180, b: 140 }),
        ("codex", RgbColor { r: 30, g: 30, b: 30 }),
    ])
}

pub fn get_color(agent: &str) -> Option<RgbColor> {
    default_colors().get(agent).copied()
}

pub fn format_agents(list: &[String]) -> String {
    list.join(", ")
}
