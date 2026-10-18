// The Go side of the window, the bridge, as a static C library linked into the executable (RMTEC).

use std::env;
use std::path::PathBuf;
use std::process::Command;

fn main() {
    let out = PathBuf::from(env::var("OUT_DIR").expect("OUT_DIR"));
    let root = PathBuf::from(env::var("CARGO_MANIFEST_DIR").expect("CARGO_MANIFEST_DIR")).join("..");
    let go = env::var("GO").unwrap_or_else(|_| "go".into());
    let lib = out.join("libpolvo.a");
    let status = Command::new(&go)
        .args(["build", "-buildmode=c-archive", "-trimpath", "-tags", "netgo,osusergo,polvo_p2p", "-o"])
        .arg(&lib)
        .arg("./cmd/libpolvo")
        .current_dir(&root)
        .status()
        .unwrap_or_else(|e| panic!("polvo: running {go} to build libpolvo: {e}"));
    assert!(status.success(), "polvo: building libpolvo failed");
    println!("cargo:rustc-link-search=native={}", out.display());
    println!("cargo:rustc-link-lib=static=polvo");
    if env::var("CARGO_CFG_TARGET_OS").as_deref() == Ok("macos") {
        println!("cargo:rustc-link-lib=framework=CoreFoundation");
        println!("cargo:rustc-link-lib=framework=Security");
    }
    for dir in ["cmd/libpolvo", "internal", "go.mod", "go.sum"] {
        println!("cargo:rerun-if-changed={}", root.join(dir).display());
    }
    println!("cargo:rerun-if-env-changed=GO");
}
