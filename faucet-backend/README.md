 # Minimalistic Oasis Testnet faucet.

 This is a minimalistic testnet faucet intended to be used to fund Oasis
 TEST tokens.  It exposes a minimalistic RESTful API over http(s), and
 has support for reCAPTCHA v2 integration.

 #### Setup

  * Create a data directory.
  * Initialize an entity via `oasis-node registry entity init` to be
    used as the funding account.  The address can be derived via
    `oasis-node stake pubkey2address --public_key <entity ID>`.
  * Fund the account.
  * Optionally create a webroot directory, and populate it with static
    assets.
  * Configure the faucet-backend (Default: `faucet-backend.toml`).
  * Run the faucet-backend.

#### API

There is one POST call. `https://host:port/api/v1/fund`.  The call
arguments are taken via the `paratime`, `account` and `amount` (in tokens)
query arguments (can also be sent in the POST form).  If configured, the
user's reCAPTCHA response MUST be sent via the `g-recaptcha-response` POST
form entry.  As a concession to testing, if the reCAPTCHA auth is not
configured, the API call will also operate via HTTP GET.  The paratime
should be specified by paratime name (`emerald` etc), and omitted or set
to empty if consensus funding is requested.

The request will respond with a trivial JSON encoded object with `result`,
containing a human readable representation of the status, and set the HTTP
status code to `OK` on success, and an error code as appropriate.
