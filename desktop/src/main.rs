// @anchors
//   code: DSMND
//   ref: JNNTJ

/// A test named by its scenario, the code and the title as the feature has them (the Rust
/// counterpart of Go's `t.Run("CODE: title", …)`).
#[cfg(test)]
macro_rules! scenario {
    ($name:ident, $title:literal, $body:block) => {
        #[doc = $title]
        #[test]
        fn $name() $body
    };
}

mod keys; // @dep: KYIMK
mod prefs; // @dep: PRIMP

fn main() {}
