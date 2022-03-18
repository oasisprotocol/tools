# unmnemonic

This tool implements wallet key derivation from a BIP-39 mnemonic using
the various algorithms that are or have been used by Oasis.

The currently supported methods are:

- ADR-0008 SLIP-0010
- Legacy pre-SLIP Oasis Ledger

It is intended to be used for the purposes of migration and/or disaster
recovery.  Use of this tool can lead to the total compromise of all accounts
associated with a given mnemonic, and it's use is heavily discouraged.

Due to the intended "for recovery" use of the tool, it explicitly refrains
from importing oasis-core or any other major dependencies in the hopes
that it will basically always work.
