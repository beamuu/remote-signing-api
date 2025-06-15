package kmsutil

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"

	"remote-signing-api/internal/firestoreutil"

	"cloud.google.com/go/firestore"
	kms "cloud.google.com/go/kms/apiv1"
	"github.com/ethereum/go-ethereum/crypto"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
)

// ---------- shared service ----------
type SignerService struct {
	KMSClient *kms.KeyManagementClient
	FSClient  *firestore.Client
	ProjectID string
}

func NewSignerService(ctx context.Context, projectID string) (*SignerService, error) {
	kc, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, err
	}
	fc, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		kc.Close()
		return nil, err
	}
	return &SignerService{KMSClient: kc, FSClient: fc, ProjectID: projectID}, nil
}
func (s *SignerService) Close() {
	s.KMSClient.Close()
	s.FSClient.Close()
}

// ---------- signing ----------
func (s *SignerService) SignWithKMS(ctx context.Context, address string, hash []byte) (string, error) {
	keyPath, err := firestoreutil.GetKMSKeyForAddress(ctx, s.FSClient, address)
	if err != nil {
		return "", err
	}

	// sign hash via KMS
	req := &kmspb.AsymmetricSignRequest{
		Name: keyPath,
		Digest: &kmspb.Digest{
			Digest: &kmspb.Digest_Sha256{Sha256: hash},
		},
	}
	resp, err := s.KMSClient.AsymmetricSign(ctx, req)
	if err != nil {
		return "", err
	}

	// parse ASN.1 → r/s
	var esig struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(resp.Signature, &esig); err != nil {
		return "", err
	}

	// fetch public key to recover v
	pubResp, err := s.KMSClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: keyPath})
	if err != nil {
		return "", err
	}
	pubDer, _ := pemToDER(pubResp.Pem)
	pubIfc, _ := x509.ParsePKIXPublicKey(pubDer)
	pub := pubIfc.(*ecdsa.PublicKey)

	sigBytes := make([]byte, 65)
	copy(sigBytes[0:32], esig.R.Bytes())
	copy(sigBytes[32:64], esig.S.Bytes())
	for v := byte(0); v < 2; v++ {
		sigBytes[64] = v
		if rec, _ := crypto.SigToPub(hash, sigBytes); rec != nil &&
			rec.X.Cmp(pub.X) == 0 && rec.Y.Cmp(pub.Y) == 0 {
			sigBytes[64] += 27
			break
		}
	}
	return "0x" + hex.EncodeToString(sigBytes), nil
}

// ---------- onboarding ----------
func (s *SignerService) CreateCustomerKey(
	ctx context.Context,
	location, keyRing, customerID string,
) (string, error) {
	parent := fmt.Sprintf("projects/%s/locations/%s/keyRings/%s", s.ProjectID, location, keyRing)
	cryptoKeyID := fmt.Sprintf("%s-%d", customerID, time.Now().UnixNano())

	ckReq := &kmspb.CreateCryptoKeyRequest{
		Parent:      parent,
		CryptoKeyId: cryptoKeyID,
		CryptoKey: &kmspb.CryptoKey{
			Purpose: kmspb.CryptoKey_ASYMMETRIC_SIGN,
			VersionTemplate: &kmspb.CryptoKeyVersionTemplate{
				Algorithm: kmspb.CryptoKeyVersion_EC_SIGN_SECP256K1_SHA256,
			},
		},
	}
	ck, err := s.KMSClient.CreateCryptoKey(ctx, ckReq)
	if err != nil {
		return "", err
	}
	keyVersion := ck.Name + "/cryptoKeyVersions/1"

	// derive Ethereum address
	pubResp, err := s.KMSClient.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{Name: keyVersion})
	if err != nil {
		return "", err
	}
	der, _ := pemToDER(pubResp.Pem)
	pkIfc, _ := x509.ParsePKIXPublicKey(der)
	pk := pkIfc.(*ecdsa.PublicKey)
	addr := crypto.PubkeyToAddress(*pk).Hex()

	// save mapping
	if err := firestoreutil.SaveMapping(ctx, s.FSClient, addr, keyVersion); err != nil {
		return "", err
	}
	return addr, nil
}
