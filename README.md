
# Remote Signing API

Remote‑signing micro‑service for Ethereum **Clique** validator keys stored in **Google Cloud KMS**.  
It allows you to:

1. **On‑board** new **users** & create a fresh secp256k1 KMS key for each of them.  
2. **Derive** the user’s Ethereum address from the KMS public key.  
3. **Persist** `address → kmsKeyPath` in Firestore.  
4. **Sign** arbitrary 32‑byte hashes (e.g. Clique block hashes) on demand.  

The codebase is MIT‑licensed open source—you’re free to run it yourself or build paid services on top.

---

## Architecture

```
                 ┌──────────────────────┐
                 │  Google KMS          │   Key Ring: clique-signer
                 └──────────────────────┘
                           ▲   (sign / getPublicKey)
                           │
     Firestore             │
     collection            │
     validator_keys        │
            ▲              │
            │ look‑up / set│
┌───────────┴──────────────┴───────────────────────┐
│              Signer API (Fiber)                  │
│  • POST /onboard  → create key, return address   │
│  • POST /sign     → sign hash with user key      │
└──────────────────────────────────────────────────┘
```

* **Key Ring** groups all validator keys (easier IAM).  
* **Firestore mapping** lets the service pick the right key at runtime.  
* A single `SignerService` instance holds **one** KMS client and **one** Firestore client for efficiency.

---

## Source Layout

```
cmd/api/             → main.go        (Fiber server, env config)
handler/             → HTTP handlers
kmsutil/             → SignerService (sign & onboarding) + PEM helper
firestoreutil/       → Get / Save address ↔ key mapping
go.mod / go.sum
Dockerfile
```

---

## Endpoints

| Method | Path        | Body (JSON)                              | Result |
|--------|-------------|------------------------------------------|--------|
| `POST` | `/onboard`  | `{ "user_id": "alice" }`                 | `{ "address": "0xABCD…EF" }` |
| `POST` | `/sign`     | `{ "address": "0xABCD…", "hash": "0x…32B" }` | `{ "signature": "0xRRSV…" }` |

---

## Environment Variables

| Variable           | Description                                   |
|--------------------|-----------------------------------------------|
| `GCP_PROJECT_ID`   | GCP project that owns KMS + Firestore         |
| `KMS_LOCATION`     | KMS location (e.g. `global` or `asia-southeast1`) |
| `KMS_KEY_RING`     | Existing key‑ring name (e.g. `clique-signer`) |

*Make sure the service account running the container has* **`roles/cloudkms.signerVerifier`** *on that key‑ring and* **`roles/datastore.user`** *for Firestore.*

---

## Quick Start (local)

```bash
# 1. Create key-ring once
gcloud kms keyrings create clique-signer --location=global

# 2. Run the API
export GCP_PROJECT_ID=my-project
export KMS_LOCATION=global
export KMS_KEY_RING=clique-signer
go run ./cmd/api
```

### Test with `curl`

```bash
# Onboard a user
curl -X POST http://localhost:8080/onboard   -H "Content-Type: application/json"   -d '{"user_id":"alice"}'

# Sign a dummy hash
curl -X POST http://localhost:8080/sign   -H "Content-Type: application/json"   -d '{"address":"0x....","hash":"0x0123..."}'
```

---

## Build & Run with Docker

```bash
# build
docker build -t clique-signer-api:latest .

# run
docker run -p 8080:8080   -e GCP_PROJECT_ID=$GCP_PROJECT_ID   -e KMS_LOCATION=global   -e KMS_KEY_RING=clique-signer   clique-signer-api:latest
```

The Dockerfile uses a **multi‑stage build** (Go 1.24.1‑alpine → tiny alpine runtime) and a BuildKit cache mount to speed up module downloads.

---

## Deploy to Cloud Run (example)

```bash
gcloud builds submit --tag gcr.io/$GCP_PROJECT_ID/clique-signer-api
gcloud run deploy clique-signer-api   --image gcr.io/$GCP_PROJECT_ID/clique-signer-api   --region asia-southeast1   --platform managed   --allow-unauthenticated   --set-env-vars GCP_PROJECT_ID=$GCP_PROJECT_ID,KMS_LOCATION=global,KMS_KEY_RING=clique-signer
```

---

## Firestore Schema

```
Collection: validator_keys
Document ID: <lower‑cased Ethereum address>
Fields:
  kms_key : "projects/<proj>/locations/<loc>/keyRings/<ring>/cryptoKeys/<key>/cryptoKeyVersions/1"
```

---

## Monetisation Ideas

Because the project is MIT‑licensed open source you’re free to:

* **Offer a hosted “signer‑as‑a‑service”** and charge per signature or monthly subscription.
* Provide paid SLAs, dashboards, or premium monitoring plugins.
* Sell support packages while the community enjoys the core for free.

---

## Security Notes

* **Onboard** returns only the derived address—private keys never leave KMS.
* Every request in `/sign` is checked against Firestore; unknown addresses fail.
* Consider adding JWT / API‑key auth and usage quotas for production.

---

## License

MIT – build cool things, give back, and feel free to monetise responsibly 🚀
