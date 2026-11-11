package config_test

import (
	"reflect"
	"strings"
	"testing"
)

// FuzzDecode feeds arbitrary bytes to the decoder (datum 0.10). It must never
// panic, and anything it accepts must survive encoding and decoding unchanged,
// so the encoder and decoder cannot drift apart.
func FuzzDecode(f *testing.F) {
	seeds := []string{
		validConfiguration,
		"",
		"schema_version: 1\npages: []\n",
		"schema_version: 1\npages:\n  - name: A\n    keys: []\n",
		strings.Replace(validConfiguration, "retries: 3", "retries: 0x3", 1),
		"&anchor a: *anchor\n",
		"schema_version: 1\nschema_version: 2\n",
		"[[[[[[[[[[",
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, fileContent []byte) {
		decoder := newTestDecoder(t)

		configuration, decodeError := decoder.Decode(fileContent)
		if decodeError != nil {
			return
		}

		encoded, encodeError := decoder.Encode(configuration)
		if encodeError != nil {
			t.Fatalf("an accepted configuration failed to encode: %v", encodeError)
		}

		roundTripped, roundTripError := decoder.Decode(encoded)
		if roundTripError != nil {
			t.Fatalf("an encoded configuration was rejected: %v\n%s", roundTripError, encoded)
		}

		if !reflect.DeepEqual(configuration, roundTripped) {
			t.Fatalf("round trip changed an accepted configuration:\n%s", encoded)
		}
	})
}
