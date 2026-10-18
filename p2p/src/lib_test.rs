// @anchors
//   code: P2TST
//   ref: PTRNS

use super::*;
use std::ffi::{CStr, CString};

fn seed(n: u8) -> [u8; 32] {
    [n; 32]
}

async fn peer(n: u8) -> Peer {
    Peer::bind(seed(n), Relay::Disabled).await.unwrap()
}

#[doc = "PTRNS-B01: O endpoint de uma instalação"]
#[tokio::test]
async fn ptrns_b01_o_endpoint_de_uma_instalacao() {
    let p = peer(1).await;
    let id = SecretKey::from_bytes(&seed(1)).public().to_string();
    let addr: serde_json::Value = serde_json::from_str(&p.addr()).unwrap();
    assert_eq!(p.id(), id);
    assert!(addr.to_string().contains(&id), "the address names the key: {addr}");
    assert!(p.ep.addr().ip_addrs().next().is_some(), "a direct address: {addr}");
}

#[doc = "PTRNS-B02: Ligar e aceitar"]
#[tokio::test]
async fn ptrns_b02_ligar_e_aceitar() {
    let (a, b) = (peer(2).await, peer(3).await);
    let server = tokio::spawn({
        let b = b.clone();
        async move {
            let mut s = b.accept().await.unwrap().unwrap();
            let got = s.recv.read_to_end(1 << 20).await.unwrap();
            s.send.write_all(&got).await.unwrap();
            s.send.finish().unwrap();
            s.send.stopped().await.ok();
        }
    });
    let mut s = a.connect(&b.addr()).await.unwrap();
    s.send.write_all(b"ola").await.unwrap();
    s.send.finish().unwrap();
    let back = s.recv.read_to_end(1 << 20).await.unwrap();
    server.await.unwrap();
    assert_eq!(back, b"ola");
}

#[doc = "PTRNS-B03: A chave do outro lado"]
#[tokio::test]
async fn ptrns_b03_a_chave_do_outro_lado() {
    let (a, b) = (peer(4).await, peer(5).await);
    let accepted = tokio::spawn({
        let b = b.clone();
        async move { b.accept().await.unwrap().unwrap().remote }
    });
    let s = a.connect(&b.addr()).await.unwrap();
    let theirs = accepted.await.unwrap();
    // b's addresses, under a key b does not have
    let mut wrong: serde_json::Value = serde_json::from_str(&b.addr()).unwrap();
    wrong["id"] = serde_json::Value::String(SecretKey::from_bytes(&seed(6)).public().to_string());
    let refused = tokio::time::timeout(std::time::Duration::from_secs(20), a.connect(&wrong.to_string())).await;
    assert_eq!((s.remote, theirs), (b.id(), a.id()));
    assert!(!matches!(refused, Ok(Ok(_))), "want the wrong key refused");
}

#[doc = "PTRNS-E01: Um endereço que não é"]
#[tokio::test]
async fn ptrns_e01_um_endereco_que_nao_e() {
    assert!(peer(7).await.connect("isto não é um endereço").await.is_err());
}

#[doc = "PTRNS-E02: Um relay que não é URL"]
#[tokio::test]
async fn ptrns_e02_um_relay_que_nao_e_url() {
    assert!(Peer::bind(seed(8), Relay::parse("não é url")).await.is_err());
}

fn take(p: *mut std::ffi::c_char) -> String {
    let s = unsafe { CStr::from_ptr(p) }.to_string_lossy().into_owned();
    unsafe { ffi::polvo_p2p_free(p) };
    s
}

#[doc = "PTRNS-B04: A API C"]
#[test]
fn ptrns_b04_a_api_c() {
    use ffi::*;
    let none = CString::new("").unwrap();
    let mut err = std::ptr::null_mut();
    let (ka, kb) = (seed(9), seed(10));
    let a = unsafe { polvo_p2p_open(ka.as_ptr(), none.as_ptr(), &mut err) };
    let b = unsafe { polvo_p2p_open(kb.as_ptr(), none.as_ptr(), &mut err) };
    assert!(a > 0 && b > 0);
    let addr = CString::new(take(polvo_p2p_addr(b))).unwrap();
    let server = std::thread::spawn(move || {
        let mut who = std::ptr::null_mut();
        let s = unsafe { polvo_p2p_accept(b, &mut who, std::ptr::null_mut()) };
        let mut buf = [0u8; 16];
        let mut got = Vec::new();
        loop {
            let n = unsafe { polvo_p2p_read(s, buf.as_mut_ptr(), buf.len()) };
            if n <= 0 {
                break;
            }
            got.extend_from_slice(&buf[..n as usize]);
        }
        unsafe { polvo_p2p_write(s, got.as_ptr(), got.len()) };
        polvo_p2p_close(s);
        take(who)
    });
    let s = unsafe { polvo_p2p_connect(a, addr.as_ptr(), &mut err) };
    assert!(s > 0);
    unsafe { polvo_p2p_write(s, b"ola".as_ptr(), 3) };
    polvo_p2p_close_write(s);
    let mut buf = [0u8; 16];
    let mut back = Vec::new();
    loop {
        let n = unsafe { polvo_p2p_read(s, buf.as_mut_ptr(), buf.len()) };
        if n <= 0 {
            break;
        }
        back.extend_from_slice(&buf[..n as usize]);
    }
    let who = server.join().unwrap();
    let waiting = std::thread::spawn(move || unsafe { polvo_p2p_accept(b, std::ptr::null_mut(), std::ptr::null_mut()) });
    std::thread::sleep(std::time::Duration::from_millis(200));
    polvo_p2p_shutdown(b);
    let after = waiting.join().unwrap();
    polvo_p2p_close(s);
    polvo_p2p_shutdown(a);
    assert_eq!((back.as_slice(), who, after), (&b"ola"[..], SecretKey::from_bytes(&ka).public().to_string(), 0));
}
