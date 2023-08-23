# attestation-tool

Tool for testing remote attestation against the Intel SGX's development server
through the Oasis Protocol Foundations IAS proxy.

[Fortanix]: https://www.fortanix.com/

## Prerequisites

To build the tool, ensure you have [Rust] and [rustup] installed on your system.
For more details, see [Oasis Core's Development Setup Prerequisites]
documentation, the Rust section.

To run the tool, ensure that your hardware has SGX support, that SGX support is
enabled and that the additional driver and software components are properly
installed and running. Namely:

- SGX kernel driver is loaded,
- SGX device is accessible by the user running the binary,
- AESM Service is running and its socket is available.

For more details, see our documentation on
[Setting up a Trusted Execution Environment (TEE)][oasis-setup-tee].

[Rust]: https://www.rust-lang.org/
[rustup]: https://rustup.rs/
[Oasis Core's Development Setup Prerequisites]:
  https://docs.oasis.io/core/development-setup/prerequisites
[oasis-setup-tee]:
  https://docs.oasis.io/node/run-your-node/prerequisites/set-up-trusted-execution-environment-tee/

## Building

To build the tool run:

```
cargo build --release
```

The binary will be located in `target/release/attestation-tool`.

## Using

To test remote attestation against the development server, simply run the
resulting binary:

```
./attestation-tool
```

_NOTE: You might need to run this as `root` user or via `sudo`._

The output may be something like the following:
```
Using IAS URL: https://iasproxy.fortanix.com/
Enclave report contents:
  CPUSVN: 13130207ff8006000000000000000000
  MISCSELECT: (empty)
  ATTRIBUTES: Attributes { flags: INIT | DEBUG | MODE64BIT, xfrm: 31 }
  MRENCLAVE: d40c35b716c9ef1715d26100bb5e152d5045543017dacfcb492697028985cb7c
  MRSIGNER: 9affcfae47b848ec2caf1c49b4b283531e1cc425f93582b36806e52a43d78d1a
  ISVPRODID: 0
  ISVSVN: 0
  KEYID: 6b424515a72b29c8f170d34102ea87c400000000000000000000000000000000
  MAC: 7a17bc433aea9d2b25c92db3b152b660
  QUOTE: AgABAEsMAAANAA0AAAAAAAHKR+rmsOoT0eFEz2yFlhIAAAAAAAAAAAAAAAAAAAAAExMCB/+ABgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABwAAAAAAAAAfAAAAAAAAANQMNbcWye8XFdJhALteFS1QRVQwF9rPy0kmlwKJhct8AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACa/8+uR7hI7CyvHEm0soNTHhzEJfk1grNoBuUqQ9eNGgAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAqAIAADxMrJLUV+AY2BbArxR0vHVeXVOdJ/c+ZAP9uli2mdyxe4RgiQ+I7r6iBqUAU8oiJCRDbQpjHLKUtSiK3SDeGT+IOh9/zXm63Yg6P/r6WDOKaCt+KFdZZmpTLJLmuT7eATHrHRWOJvmX1sCHtd7vt/ehscO1Z91bt9lga+eE0fn9Y8QiSh74Xj3ilAa9ykN2+mAOKMc7A5yHmoFqBhc6pTLrA6vGBmlD/WY7QnDidGMX7JfZKXHMhyim80N+jpavX0gbWl024PkMAmY5yS/HXFwuDjltKyJpXgcTsWKBxTxdHTbOC28HEFD8reQqibE+/uTxNZ9fmMhb7v5w8v/sJCVvwFqepaYl/rrz5Gys0pmijX/AlKqyu1gNHuOrciXyIMPeqlXktTbHrBG3pGgBAABGaStZsB7JdLouEZphC/nfqoLGz5kMUzGHrxNzs7OhT5oL/EOM30bGjUWkOGCPY4iqjnDug0egc4jLEs2LUqcX+CLsiOZ7fENM/UItyS8szlPLDKE0UcYTf0iKIKiRIdcsxocCCzkmlUesYgxtW5Vf5CBFbgmoF+JCQtVdnv1okJkt/mSfpnUT842JKBBaa4P35mkCJOpiuC/4mgwF89x78l4hssXmkaiOP4Gw5Pq0htv0CdNVrP0+TgCf9kwdO9mBmDX20eZ9C3Qa0hsGnr83XeULq0/JLOOmSAebVTWTPfJRv1qMvZz00iUKl1ovDWaQDwhgPu1ZACOel3QfHOeitWNBTG+1Vp2JVXQK9hWrAKBIRkqtkM/1HWYuk7XrdAmZRgzY1bR4igxvvuM6ClHjfRB/AcsxTR29QY7gzR8aCUf0sm1U+t9mRrnZvVJimPUJoV1aN4bJrYukpcbx3lG2dMnvSkvN77pYQhiNtguaYiEqms5VuE2E

IAS report contents:
  id: 262963481655852044638179126525428053886
  version: 4
  isv_enclave_quote_status: SwHardeningNeeded
  isv_enclave_quote_body header
    version: 2
    signature type: 1
    gid: 4b0c0000
    isvsvn qe: 13
    isvsvn pce: 13
    basename: 01ca47eae6b0ea13d1e144cf6c85961200000000000000000000000000000000
  revocation_resason: None
  pse_manifest_status: None
  pse_manifest_hash: None
  platform_info_blob: None
  nonce: None
  epid_pseudonym: b8c54f2bfbe3f9d2b31d16c52b634d8c23e5c53a2bb911d55071f72f91347d08bb4cbd76f60571cd49244392df02008a2319e835895ab174a74a3fb8e94ef0fba5c182b0de1ebb52015148d46bda4c4fda8b065dfec1d9363d1e882055576fb9ce4e55898884b6c527afbe5e4f1d339e4b642b5861b3bdc3771366afcbb9890e

Advisory URL: https://security-center.intel.com
Advisory IDs: INTEL-SA-00334, INTEL-SA-00615
```

### Success

In case the report omits the `Platform status` section near the end, then
your attestation was successful.

### Failure

In case the report containes the `Platform status` section near the end, then
your attestation was NOT successful.

If you get something like:

```
Platform status:
  Ok(V2(QE_EPID_GROUP_OUT_OF_DATE | QUOTE_CPUSVN_OUT_OF_DATE))
```

your system's platform is out of date and the BIOS likely needs to be upgraded.

If you get something like:

```
Platform status:
  Ok(V2(PLATFORM_CONFIGURATION_NEEDED))
```

your system's platform is updated but your BIOS configuration needs to be
changed.
Most likely you need to disable hyperthreading and/or any
overclocking/undervolting options (e.g. SpeedStep).

## FAQ

### Why are there `Advisory IDs` present at the end of the report even if the system's platform is up-to-date?

Because the status is `SwHardeningNeeded` (i.e. software hardenning needed), IAS
cannot know if you have software mitigations applied in your enclave code or
not.
