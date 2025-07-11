package ed25519

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"testing"
)

var (
	vectors = []struct {
		sk, pk, alpha, h, k, u, v, pi, beta []byte
	}{
		{
			sk:    dh("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60"),
			pk:    dh("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a"),
			alpha: []byte(""),
			h:     dh("91bbed02a99461df1ad4c6564a5f5d829d0b90cfc7903e7a5797bd658abf3318"),
			k:     dh("8a49edbd1492a8ee09766befe50a7d563051bf3406cbffc20a88def030730f0f"),
			u:     dh("aef27c725be964c6a9bf4c45ca8e35df258c1878b838f37d9975523f09034071"),
			v:     dh("5016572f71466c646c119443455d6cb9b952f07d060ec8286d678615d55f954f"),
			pi:    dh("8657106690b5526245a92b003bb079ccd1a92130477671f6fc01ad16f26f723f26f8a57ccaed74ee1b190bed1f479d9727d2d0f9b005a6e456a35d4fb0daab1268a1b0db10836d9826a528ca76567805"),
			beta:  dh("90cf1df3b703cce59e2a35b925d411164068269d7b2d29f3301c03dd757876ff66b71dda49d2de59d03450451af026798e8f81cd2e333de5cdf4f3e140fdd8ae"),
		},
		{
			sk:    dh("4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb"),
			pk:    dh("3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c"),
			alpha: dh("72"),
			h:     dh("5b659fc3d4e9263fd9a4ed1d022d75eaacc20df5e09f9ea937502396598dc551"),
			k:     dh("d8c3a66921444cb3427d5d989f9b315aa8ca3375e9ec4d52207711a1fdb44107"),
			u:     dh("1dcb0a4821a2c48bf53548228b7f170962988f6d12f5439f31987ef41f034ab3"),
			v:     dh("fd03c0bf498c752161bae4719105a074630a2aa5f200ff7b3995f7bfb1513423"),
			pi:    dh("f3141cd382dc42909d19ec5110469e4feae18300e94f304590abdced48aed5933bf0864a62558b3ed7f2fea45c92a465301b3bbf5e3e54ddf2d935be3b67926da3ef39226bbc355bdc9850112c8f4b02"),
			beta:  dh("eb4440665d3891d668e7e0fcaf587f1b4bd7fbfe99d0eb2211ccec90496310eb5e33821bc613efb94db5e5b54c70a848a0bef4553a41befc57663b56373a5031"),
		},
		{
			sk:    dh("c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7"),
			pk:    dh("fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025"),
			alpha: dh("af82"),
			h:     dh("bf4339376f5542811de615e3313d2b36f6f53c0acfebb482159711201192576a"),
			k:     dh("5ffdbc72135d936014e8ab708585fda379405542b07e3bd2c0bd48437fbac60a"),
			u:     dh("2bae73e15a64042fcebf062abe7e432b2eca6744f3e8265bc38e009cd577ecd5"),
			v:     dh("88cba1cb0d4f9b649d9a86026b69de076724a93a65c349c988954f0961c5d506"),
			pi:    dh("9bc0f79119cc5604bf02d23b4caede71393cedfbb191434dd016d30177ccbf8096bb474e53895c362d8628ee9f9ea3c0e52c7a5c691b6c18c9979866568add7a2d41b00b05081ed0f58ee5e31b3a970e"),
			beta:  dh("645427e5d00c62a23fb703732fa5d892940935942101e456ecca7bb217c61c452118fec1219202a0edcf038bb6373241578be7217ba85a2687f7a0310b2df19f"),
		},
	}
)

func TestEncodeToCurve(t *testing.T) {
	for _, v := range vectors {
		H := encodeToCurve(v.pk, v.alpha)
		if got, want := H.Bytes(), v.h; !bytes.Equal(got, want) {
			t.Errorf("got = %x, want = %x", got, want)
		}
	}
}

// func TestGenerateNonce(t *testing.T) {
// 	for _, v := range vectors {
// 		if got, want := generateNonce(v.sk, v.h).Bytes(), v.k; !bytes.Equal(got, want) {
// 			t.Errorf("got = %x, want = %x", got, want)
// 		}
// 	}
// }

func TestEvaluate(t *testing.T) {
	for _, v := range vectors {
		signer, err := NewVRFSigner(v.sk)
		if err != nil {
			t.Error(err)
		}
		index, proof := signer.ECVRFProve(v.alpha)
		if got, want := index[:], v.beta[:32]; !bytes.Equal(got, want) {
			t.Errorf("got = %x, want = %x", got, want)
		}
		if got, want := proof, v.pi; !bytes.Equal(got, want) {
			t.Errorf("got = %x, want = %x", got, want)
		}
	}
}

func TestProofToHash(t *testing.T) {
	for _, v := range vectors {
		verifier, err := NewVRFVerifier(v.pk)
		if err != nil {
			t.Error(err)
		}
		index, err := verifier.ECVRFVerify(v.alpha, v.pi)
		if err != nil {
			t.Error(err)
		}
		if got, want := index[:], v.beta[:32]; !bytes.Equal(got, want) {
			t.Errorf("got = %x, want = %x", got, want)
		}
	}
}

func TestProofToHashFails(t *testing.T) {
	for _, v := range vectors {
		verifier, err := NewVRFVerifier(v.pk)
		if err != nil {
			t.Error(err)
		}
		_, err = verifier.ECVRFVerify([]byte("a"), v.pi)
		if got, want := err, ErrInvalidVRF; got != want {
			t.Errorf("got = %v, want = %v", got, want)
		}
		for i := 0; i < len(v.pi); i++ {
			pi := make([]byte, len(v.pi))
			copy(pi, v.pi)
			pi[i] ^= 1
			_, err = verifier.ECVRFVerify(v.alpha, pi)
			if got, want := err, ErrInvalidVRF; got != want {
				t.Errorf("got = %v, want = %v", got, want)
			}
		}
	}
}

var testHashValueAsPointInvalidParameters = []struct {
	hash []byte
}{
	// Length of hash is not 32
	{[]byte{0}},

	// Non-canonical encodings
	{[]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{253, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{252, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{251, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{247, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{246, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{243, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{242, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{241, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},
	{[]byte{240, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}},

	{[]byte{255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{253, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{252, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{251, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{247, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{246, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{243, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{242, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{241, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
	{[]byte{240, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255}},
}

func TestInterpretHashValueAsPointInvalid(t *testing.T) {
	for _, p := range testHashValueAsPointInvalidParameters {
		point := interpretHashValueAsPoint(p.hash)

		if point != nil {
			t.Fatal("expected no point")
		}
	}
}

func TestInterpretHashValueAsPointValid(t *testing.T) {
	hash := []byte{1, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 127}
	point := interpretHashValueAsPoint(hash)

	if point == nil {
		t.Fatal("expected a point")
	}
}

func BenchmarkEvaluate(b *testing.B) {
	priv := make([]byte, 32)
	rand.Read(priv)

	signer, err := NewVRFSigner(priv)
	if err != nil {
		b.Fatal(err)
	}
	m := make([]byte, 32)
	rand.Read(m)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signer.ECVRFProve(m)
	}
}

func dh(h string) []byte {
	result, err := hex.DecodeString(h)
	if err != nil {
		panic("DecodeString failed")
	}
	return result
}
