// @anchors
//   code: P2IMP
//   ref: PTRNS

//! Polvo's transport between networks (PTRNS): an iroh endpoint whose address is the installation's
//! ed25519 key. One QUIC stream a connection carries the bytes of the SSH that runs on top, as it runs on
//! TCP. The Rust API serves the desktop (and the mobile app); `ffi` gives the same to the node in Go.

use iroh::endpoint::{Connection, RecvStream, SendStream, presets};
use iroh::{Endpoint, EndpointAddr, RelayMap, RelayMode, RelayUrl, SecretKey};

/// PTRNS-B04: the C API, for the node in Go.
pub mod ffi; // @dep: P2FFI

/// The protocol of a Polvo connection, for the endpoint to accept only these (PTRNS-B01).
pub const ALPN: &[u8] = b"polvo/ssh/1";

/// What a connection is for, the first byte the side that connects writes: a QUIC stream reaches the
/// other side only with its first bytes, and the one that accepts must not wait for the other to speak
/// (PTRNS-B02). `SSH` carries the SSH of Polvo; `ADDR` the new address of an installation (PTRNS-B06).
pub const SSH: u8 = b'P';
pub const ADDR: u8 = b'A';

/// How long a closed stream of the C API waits for the other side to have what was written (PTRNS-B04).
pub const CLOSE_WAIT: std::time::Duration = std::time::Duration::from_secs(5);

/// Which relays the endpoint uses to get through NATs, when a direct path does not work.
#[derive(Clone, Debug, PartialEq)]
pub enum Relay {
    /// The public relays of iroh.
    Default,
    /// None: only direct paths (the same network, or a reachable address).
    Disabled,
    /// This relay.
    Url(String),
}

impl Relay {
    /// The relay from text: "default", "" (none), or a relay's URL.
    pub fn parse(text: &str) -> Relay {
        match text {
            "default" => Relay::Default,
            "" => Relay::Disabled,
            url => Relay::Url(url.to_string()),
        }
    }
}

/// The endpoint of this installation.
#[derive(Clone)]
pub struct Peer {
    ep: Endpoint,
}

/// A connection from or to another installation: its key, and the stream both ways.
pub struct Stream {
    /// The key of the other installation, which the handshake proved (PTRNS-B03).
    pub remote: String,
    /// What the connection is for: `SSH` or `ADDR` (PTRNS-B06).
    pub kind: u8,
    pub send: SendStream,
    pub recv: RecvStream,
    // the stream lives while its connection does
    _conn: Connection,
}

impl Peer {
    /// The endpoint of the installation whose key's secret is `seed` (PTRNS-B01), on the UDP `port` of
    /// every network (0: any); on another port when that one is taken (PTRNS-B05).
    pub async fn bind(seed: [u8; 32], relay: Relay, port: u16) -> Result<Peer, String> {
        match Self::bind_on(seed, relay.clone(), port).await {
            Err(_) if port != 0 => Self::bind_on(seed, relay, 0).await,
            bound => bound,
        }
    }

    async fn bind_on(seed: [u8; 32], relay: Relay, port: u16) -> Result<Peer, String> {
        let mode = match relay {
            Relay::Default => RelayMode::Default,
            Relay::Disabled => RelayMode::Disabled,
            Relay::Url(url) => match url.parse::<RelayUrl>() {
                Ok(url) => RelayMode::Custom(RelayMap::from(url)),
                // PTRNS-E02
                Err(e) => return Err(format!("p2p: the relay {url} is not a URL: {e}")),
            },
        };
        let builder = Endpoint::builder(presets::Minimal)
            .secret_key(SecretKey::from_bytes(&seed))
            .alpns(vec![ALPN.to_vec()])
            .relay_mode(mode)
            .clear_ip_transports();
        let builder = match builder.bind_addr(std::net::SocketAddr::from(([0, 0, 0, 0], port))) {
            Ok(b) => b,
            // PTRNS-E02
            Err(e) => return Err(format!("p2p: the port {port}: {e}")),
        };
        // IPv6 when the machine has it; an endpoint with only IPv4 is still one
        let builder = match builder.bind_addr(std::net::SocketAddr::from(([0u16; 8], port))) {
            Ok(b) => b,
            Err(e) => return Err(format!("p2p: the port {port}: {e}")),
        };
        let bound = builder.bind().await;
        match bound {
            Ok(ep) => Ok(Peer { ep }),
            // PTRNS-E02
            Err(e) => Err(format!("p2p: the endpoint did not start: {e}")),
        }
    }

    /// The key of this installation, which is its address.
    pub fn id(&self) -> String {
        self.ep.id().to_string()
    }

    /// Where this installation is reached now: its key, the direct addresses and the relay, as JSON (PTRNS-B01).
    pub fn addr(&self) -> String {
        serde_json::to_string(&self.ep.addr()).unwrap_or_default()
    }

    /// The next connection of another installation; none when the endpoint is closed (PTRNS-B02).
    pub async fn accept(&self) -> Option<Result<Stream, String>> {
        let incoming = self.ep.accept().await?;
        Some(
            async {
                let conn = incoming.await.map_err(|e| format!("p2p: the connection failed: {e}"))?;
                let (send, mut recv) = conn.accept_bi().await.map_err(|e| format!("p2p: no stream: {e}"))?;
                let mut hello = [0u8; 1];
                recv.read_exact(&mut hello).await.map_err(|e| format!("p2p: no hello: {e}"))?;
                Ok(Stream { remote: conn.remote_id().to_string(), kind: hello[0], send, recv, _conn: conn })
            }
            .await,
        )
    }

    /// A connection to the installation at `addr` (as `addr` gives it); an error when it cannot be reached,
    /// or another key answers (PTRNS-E01, PTRNS-B03).
    pub async fn connect(&self, addr: &str) -> Result<Stream, String> {
        self.connect_for(addr, SSH).await
    }

    /// A connection for `kind` (PTRNS-B06), as `connect` makes one for the SSH.
    pub async fn connect_for(&self, addr: &str, kind: u8) -> Result<Stream, String> {
        let addr: EndpointAddr = match serde_json::from_str(addr) {
            Ok(a) => a,
            // PTRNS-E01
            Err(e) => return Err(format!("p2p: {addr} is not an address: {e}")),
        };
        let conn = match self.ep.connect(addr, ALPN).await {
            Ok(c) => c,
            // PTRNS-E01: no path reaches it, or another key answers (PTRNS-B03)
            Err(e) => return Err(format!("p2p: no connection: {e}")),
        };
        let (mut send, recv) = conn.open_bi().await.map_err(|e| format!("p2p: no stream: {e}"))?;
        send.write_all(&[kind]).await.map_err(|e| format!("p2p: no hello: {e}"))?;
        Ok(Stream { remote: conn.remote_id().to_string(), kind, send, recv, _conn: conn })
    }

    /// Ends the endpoint and its connections.
    pub async fn close(&self) {
        self.ep.close().await;
    }
}

#[cfg(test)]
#[path = "lib_test.rs"]
mod tests;
