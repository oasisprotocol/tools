use std::{ffi::CString, str};

use oasis_core_runtime::common::sgx::pcs::{QEIdentity, TCBInfo, TCBLevel, TCBStatus};

use aesm_client::AesmClient;
use anyhow::{anyhow, Result};
use byteorder::{ByteOrder, LittleEndian};
use dcap_ql::quote::Qe3CertDataPckCertChain;
use dcap_ql::quote::{Quote, Quote3SignatureEcdsaP256};
use mbedtls::{alloc::List as MbedtlsList, x509::certificate::Certificate};
use rustc_hex::FromHex;
use sgx_isa::{Report, Targetinfo};
use sgxs_loaders::isgx::Device as IsgxDevice;

// Intel's PCS signing root certificate.
const PCS_TRUST_ROOT_CERT: &str = r#"-----BEGIN CERTIFICATE-----
MIICjzCCAjSgAwIBAgIUImUM1lqdNInzg7SVUr9QGzknBqwwCgYIKoZIzj0EAwIw
aDEaMBgGA1UEAwwRSW50ZWwgU0dYIFJvb3QgQ0ExGjAYBgNVBAoMEUludGVsIENv
cnBvcmF0aW9uMRQwEgYDVQQHDAtTYW50YSBDbGFyYTELMAkGA1UECAwCQ0ExCzAJ
BgNVBAYTAlVTMB4XDTE4MDUyMTEwNDUxMFoXDTQ5MTIzMTIzNTk1OVowaDEaMBgG
A1UEAwwRSW50ZWwgU0dYIFJvb3QgQ0ExGjAYBgNVBAoMEUludGVsIENvcnBvcmF0
aW9uMRQwEgYDVQQHDAtTYW50YSBDbGFyYTELMAkGA1UECAwCQ0ExCzAJBgNVBAYT
AlVTMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEC6nEwMDIYZOj/iPWsCzaEKi7
1OiOSLRFhWGjbnBVJfVnkY4u3IjkDYYL0MxO4mqsyYjlBalTVYxFP2sJBK5zlKOB
uzCBuDAfBgNVHSMEGDAWgBQiZQzWWp00ifODtJVSv1AbOScGrDBSBgNVHR8ESzBJ
MEegRaBDhkFodHRwczovL2NlcnRpZmljYXRlcy50cnVzdGVkc2VydmljZXMuaW50
ZWwuY29tL0ludGVsU0dYUm9vdENBLmRlcjAdBgNVHQ4EFgQUImUM1lqdNInzg7SV
Ur9QGzknBqwwDgYDVR0PAQH/BAQDAgEGMBIGA1UdEwEB/wQIMAYBAf8CAQEwCgYI
KoZIzj0EAwIDSQAwRgIhAOW/5QkR+S9CiSDcNoowLuPRLsWGf/Yi7GSX94BgwTwg
AiEA4J0lrHoMs+Xo5o/sX6O9QWxHRAvZUGOdRQ7cvqRXaqI=
-----END CERTIFICATE-----"#;
lazy_static::lazy_static! {
    static ref PCS_TRUST_ROOT: MbedtlsList<Certificate> = {
        let mut cert_chain = MbedtlsList::new();
        let raw_cert = CString::new(PCS_TRUST_ROOT_CERT.as_bytes()).unwrap();
        let cert = Certificate::from_pem(raw_cert.as_bytes_with_nul()).unwrap();
        cert_chain.push(cert);

        cert_chain
    };
}

// OIDs for PCK X509 certificate extensions.
const PCK_SGX_EXTENSIONS_PPID_OID: &[u64] = &[1, 2, 840, 113741, 1, 13, 1, 1];
const PCK_SGX_EXTENSIONS_OID: &[u64] = &[1, 2, 840, 113741, 1, 13, 1];
const PCK_SGX_EXTENSIONS_FMSPC_OID: &[u64] = &[1, 2, 840, 113741, 1, 13, 1, 4];
const PCK_SGX_EXTENSIONS_TCB_OID: &[u64] = &[1, 2, 840, 113741, 1, 13, 1, 2];

// TCB url. // TODO: configurable.
const TCB_URL: &str = "https://api.trustedservices.intel.com/sgx/certification/v4/tcb";
const QE_IDENTITY_URL: &str =
    "https://api.trustedservices.intel.com/sgx/certification/v4/qe/identity";

#[derive(Clone, Debug, Default, serde::Deserialize)]
/// Response from TCBInfo API.
struct TCBInfoResponse {
    #[serde(rename = "tcbInfo")]
    pub tcb_info: TCBInfo,
}

// Taken from TCBLevel.matches (which is a private function).
pub fn tcb_matches(tcb_level: &TCBLevel, tcb_comp_svn: &[u32], pcesvn: u32) -> bool {
    // a) Compare all of the SGX TCB Comp SVNs retrieved from the SGX PCK Certificate (from 01 to
    //    16) with the corresponding values in the TCB Level. If all SGX TCB Comp SVNs in the
    //    certificate are greater or equal to the corresponding values in TCB Level, go to b,
    //    otherwise move to the next item on TCB Levels list.
    for (i, comp) in tcb_level.tcb.sgx_components.iter().enumerate() {
        // At least one SVN is lower, no match.
        if tcb_comp_svn[i] < comp.svn {
            return false;
        }
    }

    // b) Compare PCESVN value retrieved from the SGX PCK certificate with the corresponding value
    //    in the TCB Level. If it is greater or equal to the value in TCB Level, read status
    //    assigned to this TCB level. Otherwise, move to the next item on TCB Levels list.
    if tcb_level.tcb.pcesvn < pcesvn {
        return false;
    }

    // Match.
    true
}

#[derive(Clone, Debug, Default, serde::Deserialize)]
struct QEIdentityResponse {
    #[serde(rename = "enclaveIdentity")]
    pub enclave_identity: QEIdentity,
}

// Taken from QEIdentity.verify.
fn qe_verify(qe: &QEIdentity, report: &Report) -> Result<()> {
    // Verify if MRSIGNER field retrieved from SGX Enclave Report is equal to the value of
    // mrsigner field in QE Identity.
    let expected_mr_signer: Vec<u8> = qe
        .mr_signer
        .from_hex()
        .map_err(|_| anyhow!("malformed QE MRSIGNER"))?;
    if expected_mr_signer != report.mrsigner {
        return Err(anyhow!("TCB verification failed: QE MRSIGNER mismatch"));
    }

    // Verify if ISVPRODID field retrieved from SGX Enclave Report is equal to the value of
    // isvprodid field in QE Identity.
    if qe.isv_prod_id != report.isvprodid {
        return Err(anyhow!("TCB verification failed: QE ISVPRODID mismatch"));
    }

    // Apply miscselectMask (binary mask) from QE Identity to MISCSELECT field retrieved from
    // SGX Enclave Report. Verify if the outcome (miscselectMask & MISCSELECT) is equal to the
    // value of miscselect field in QE Identity.
    let raw_miscselect: Vec<u8> = qe
        .miscselect
        .from_hex()
        .map_err(|err| anyhow!("malformed QE miscselect: {}", err))?;
    if raw_miscselect.len() != 4 {
        return Err(anyhow!("malformed QE miscselect"));
    }
    let raw_miscselect_mask: Vec<u8> = qe
        .miscselect_mask
        .from_hex()
        .map_err(|err| anyhow!("malformed QE miscselect mask: {}", err))?;
    if raw_miscselect_mask.len() != 4 {
        return Err(anyhow!("malformed QE miscselect"));
    }
    let expected_miscselect = LittleEndian::read_u32(&raw_miscselect);
    let miscselect_mask = LittleEndian::read_u32(&raw_miscselect_mask);
    if report.miscselect.bits() & miscselect_mask != expected_miscselect {
        return Err(anyhow!("TCB verification failed: QE MISCSELECT mismatch"));
    }

    // Apply attributesMask (binary mask) from QE Identity to ATTRIBUTES field retrieved from
    // SGX Enclave Report. Verify if the outcome (attributesMask & ATTRIBUTES) is equal to the
    // value of attributes field in QE Identity.
    let raw_attributes: Vec<u8> = qe
        .attributes
        .from_hex()
        .map_err(|err| anyhow!("malformed QE attributes: {}", err))?;
    if raw_attributes.len() != 16 {
        return Err(anyhow!("malformed QE attributes"));
    }
    let raw_attributes_mask: Vec<u8> = qe
        .attributes_mask
        .from_hex()
        .map_err(|err| anyhow!("malformed QE attributes mask: {}", err))?;
    if raw_attributes_mask.len() != 16 {
        return Err(anyhow!("malformed QE attributes mask"));
    }
    let expected_flags = LittleEndian::read_u64(&raw_attributes[..8]);
    let expected_xfrm = LittleEndian::read_u64(&raw_attributes[8..]);
    let flags_mask = LittleEndian::read_u64(&raw_attributes_mask[..8]);
    let xfrm_mask = LittleEndian::read_u64(&raw_attributes_mask[8..]);
    if report.attributes.flags.bits() & flags_mask != expected_flags {
        return Err(anyhow!(
            "TCB verification failed: QE ATTRIBUTES mismatch: flags"
        ));
    }
    if report.attributes.xfrm & xfrm_mask != expected_xfrm {
        return Err(anyhow!(
            "TCB verification failed: QE ATTRIBUTES mismatch: xfrm"
        ));
    }

    // Determine a TCB status of the Quoting Enclave.
    //
    // Go over the list of TCB Levels (descending order) and find the one that has ISVSVN that
    // is lower or equal to the ISVSVN value from SGX Enclave Report.
    if let Some(level) = qe
        .tcb_levels
        .iter()
        .find(|level| level.tcb.isv_svn <= report.isvsvn)
    {
        // Ensure that the TCB is up to date.
        if level.status == TCBStatus::UpToDate {
            return Ok(());
        }
    }

    Err(anyhow!("TCB verification failed: QE TCB out of date"))
}

fn get_algorithm_id_from_key(key_id: &Vec<u8>) -> u32 {
    const ALGORITHM_OFFSET: usize = 154;

    let mut bytes: [u8; 4] = Default::default();
    bytes.copy_from_slice(&key_id[ALGORITHM_OFFSET..ALGORITHM_OFFSET + 4]);
    u32::from_le_bytes(bytes)
}

pub fn try_ecdsa(aesm_client: &AesmClient, loader: &mut IsgxDevice) -> Result<TCBStatus> {
    const SGX_QL_ALG_ECDSA_P256: u32 = 2;

    // Fetch the first supported ECDSA key.
    let ecdsa_key_id = aesm_client
        .get_supported_att_key_ids()
        .map_err(|err| anyhow!("error obtaining attestation key: {}", err))?
        .into_iter()
        .find(|key_id| SGX_QL_ALG_ECDSA_P256 == get_algorithm_id_from_key(key_id))
        .ok_or(anyhow!("no ecdsa key"))?;

    // Fetch target info.
    // Even if ECDSA is not supported the key can be there: https://github.com/intel/linux-sgx/issues/536
    let quote_info = aesm_client
        .init_quote_ex(ecdsa_key_id.clone())
        // If this fails with 'AesmCode(UnexpectedError_1)' then ECDSA is likely not supported on this platform.
        .map_err(|err| {
            anyhow!(
                "error initializing quote for ECDSA key (ECDSA unsupported?): {}",
                err
            )
        })?;
    let ti = Targetinfo::try_copy_from(quote_info.target_info()).unwrap();
    let report = report_test::report(&ti, loader).unwrap();

    // Obtain remote attestation quote from QE.
    let res = aesm_client
        .get_quote_ex(ecdsa_key_id, report.as_ref().to_owned(), None, vec![0; 16])
        .map_err(|err| anyhow!("error obtaining attestation quote: {}", err))?;
    let quote = Quote::parse(res.quote())
        .map_err(|err| anyhow!("error parsing attestation quote: {}", err))?;

    // Verify quote signature.
    let sig = quote
        .signature::<Quote3SignatureEcdsaP256>()
        .map_err(|err| anyhow!("quote signature type not supported: {}", err))?;
    sig.verify_quote_signature(&res.quote())
        .map_err(|err| anyhow!("error verifying quote signature: {}", err))?;

    // Obtain PCK certificate chain.
    let certs = sig
        .certification_data::<Qe3CertDataPckCertChain>()
        .map_err(|err| anyhow!("only PCK certificate chain is supported: {}", err))?
        .certs;

    // Verify certificate chain.
    let mut cert_chain = MbedtlsList::new();
    for raw_cert in &certs {
        let raw_cert = CString::new(raw_cert.as_ref())?;
        let cert = Certificate::from_pem(raw_cert.as_bytes_with_nul())?;
        cert_chain.push(cert);
    }
    Certificate::verify(&cert_chain, &PCS_TRUST_ROOT, None)?;

    // Extract TCB parameters from the PCK certificate.
    let pck_cert = cert_chain.pop_front().unwrap();
    let sgx_extensions = pck_cert
        .extensions()?
        .into_iter()
        .find(|ext| ext.oid.as_ref() == PCK_SGX_EXTENSIONS_OID)
        .ok_or(anyhow!(
            "TCB verification failed: missing SGX certificate extensions"
        ))?;
    let mut ppid: Option<Vec<u8>> = None;
    let mut fmspc: Option<Vec<u8>> = None;
    let mut tcb_comp_svn: Option<[u32; 16]> = None;
    let mut pcesvn: Option<u32> = None;
    yasna::parse_der(&sgx_extensions.value, |reader| {
        reader.read_sequence_of(|reader| {
            reader.read_sequence(|reader| {
                match reader.next().read_oid()?.as_ref() {
                    PCK_SGX_EXTENSIONS_PPID_OID => {
                        // PPID
                        let raw_ppid = reader.next().read_bytes()?;
                        if raw_ppid.len() != 16 {
                            return Err(yasna::ASN1Error::new(yasna::ASN1ErrorKind::Invalid));
                        }
                        ppid = Some(raw_ppid);
                    }
                    PCK_SGX_EXTENSIONS_FMSPC_OID => {
                        // FMSPC
                        let raw_fmspc = reader.next().read_bytes()?;
                        if raw_fmspc.len() != 6 {
                            return Err(yasna::ASN1Error::new(yasna::ASN1ErrorKind::Invalid));
                        }
                        fmspc = Some(raw_fmspc);
                    }
                    PCK_SGX_EXTENSIONS_TCB_OID => {
                        // TCB
                        reader.next().read_sequence_of(|reader| {
                            reader.read_sequence(|reader| {
                                let comp_id = *reader.next().read_oid()?.as_ref().last().unwrap();
                                if (1..=16).contains(&comp_id) {
                                    // TCB Component SVNs
                                    tcb_comp_svn.get_or_insert([0; 16])[(comp_id - 1) as usize] =
                                        reader.next().read_u32()?;
                                } else if comp_id == 17 {
                                    // PCESVN
                                    pcesvn = Some(reader.next().read_u32()?);
                                } else if comp_id == 18 {
                                    // CPUSVN
                                    reader.next().read_bytes()?;
                                }
                                Ok(())
                            })
                        })?;
                    }
                    _ => {
                        reader.next().read_der()?;
                    }
                }

                Ok(())
            })
        })
    })
    .map_err(|err| anyhow!("malformed PCK certificate: {}", err))?;

    for (name, exists) in [
        ("ppid", ppid.is_some()),
        ("fmspc", fmspc.is_some()),
        ("tcb_comp_svn", tcb_comp_svn.is_some()),
        ("pcesvn", pcesvn.is_some()),
    ] {
        if !exists {
            return Err(anyhow!("TCB verification failed: missing {}", name));
        }
    }
    let ppid = ppid.unwrap();
    let fmspc = fmspc.unwrap();
    let tcb_comp_svn = tcb_comp_svn.unwrap();
    let pcesvn = pcesvn.unwrap();

    // Fetch TCB info from PCS.
    let url = format!(
        "{TCB_URL}?fmspc={fmspc_hex}&update=early",
        fmspc_hex = hex::encode(&fmspc)
    );
    println!("Using PCCS URL: {:?}", url);
    let response = ureq::get(&url).call()?;
    // XXX: tcb_cert_chain ignored.
    let _tcb_cert_chain = response.header("tcb-info-issuer-chain").map(String::from);
    let tcb = response
        .into_string()
        .map_err(|err| anyhow!("error fetching TCB info: {}", err))?;
    let tcb: TCBInfoResponse =
        serde_json::from_str(&tcb).map_err(|err| anyhow!("error parsing TCB info: {}", err))?;
    let tcb = tcb.tcb_info;

    // Validate TCB info.
    let expected_fmspc: Vec<u8> = tcb
        .fmspc
        .from_hex()
        .map_err(|err| anyhow!("tcb fmspc: {}", err))?;
    if fmspc != expected_fmspc {
        return Err(anyhow!("TCB verification failed: FMSPC mismatch"));
    }
    // Find first matching TCB level.
    let level = tcb
        .tcb_levels
        .iter()
        .find(|level| tcb_matches(level, &tcb_comp_svn, pcesvn))
        .ok_or(anyhow!(
            "Out of date TCB: SVN: {:?}, PCESVN: {:?}",
            tcb_comp_svn,
            pcesvn
        ))?
        .clone();

    println!(
        "TCB evaluation data number: {:?}",
        tcb.tcb_evaluation_data_number
    );
    println!("PPID: {:?}", hex::encode(ppid));
    println!("FMSPC: {:?}", tcb.fmspc);
    println!("System PCSEVN: {:?}", pcesvn);
    println!("TCB component SVN: {:?}", tcb_comp_svn);
    println!("TCB level PCSESVN: {:?}", level.tcb.pcesvn);
    println!("SGX components: {:?}", level.tcb.sgx_components);
    println!("TCB status: {:?}", level.status);
    println!("TCB Advisory IDs: {:?}", level.advisory_ids);

    println!("Verifying Quoting Enclave");
    // Parse QE3 report.
    let qe3_report = sig.qe3_report();
    let mut report = Vec::with_capacity(Report::UNPADDED_SIZE);
    report.extend(qe3_report);
    report.resize_with(Report::UNPADDED_SIZE, Default::default);
    let report = Report::try_copy_from(&report).ok_or(anyhow!("could not construct QE3 report"))?;

    // Fetch QE identity from PCS.
    let url = format!("{QE_IDENTITY_URL}?update=early");
    println!("Using PCCS URL: {:?}", url);
    let response = ureq::get(&url).call()?;
    let qe_identity: QEIdentityResponse = serde_json::from_str(&response.into_string()?)
        .map_err(|err| anyhow!("error parsing QE identity: {}", err))?;
    let qe_identity = qe_identity.enclave_identity;

    // Verify QE report.
    qe_verify(&qe_identity, &report)?;
    println!("Quoting Enclave: verified and up to date");

    // Everything successful return the TCB status.
    Ok(level.status)
}
