// Package xval provides a method for decrypting Xbox 360 X-value data.
package xval

import (
	"bytes"
	"crypto/des"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const FlagNone = 0
const (
	FlagAuthExFailure           = 1 << iota
	FlagAuthExNoTable           = 1 << iota
	FlagAuthExReserved          = 1 << iota
	FlagInvalidDVDGeometry      = 1 << iota
	FlagInvalidDVDDMI           = 1 << iota
	FlagDVDKeyvaultPairMismatch = 1 << iota
	FlagCRLDataInvalid          = 1 << iota
	FlagCRLCertificateRevoked   = 1 << iota
	FlagUnauthorizedInstall     = 1 << iota
	FlagKeyvaultPolicyViolation = 1 << iota
	FlagConsoleBanned           = 1 << iota
	FlagODDViolation            = 1 << iota
)

func getFlagDescriptions() map[int32]string {
	return map[int32]string{
		FlagAuthExFailure:           "AuthEx challenge failure (AP 2.5 related)",
		FlagAuthExNoTable:           "AuthEx table missing (AP 2.5 related",
		FlagAuthExReserved:          "AuthEx reserved flag (AP 2.5 related)",
		FlagInvalidDVDGeometry:      "Invalid DVD geometry",
		FlagInvalidDVDDMI:           "Invalid DVD DMI",
		FlagDVDKeyvaultPairMismatch: "Invalid CRL data",
		FlagCRLDataInvalid:          "CRL certificate revoked",
		FlagCRLCertificateRevoked:   "CRL certificate revoked",
		FlagUnauthorizedInstall:     "Unauthorized install",
		FlagKeyvaultPolicyViolation: "Keyvault policy violation",
		FlagConsoleBanned:           "Console is banned",
		FlagODDViolation:            "ODD violation",
	}
}

// Decrypt returns the decryption key, decrypted xval data, and an error
// which shows whether or not the decryption was successful.
func Decrypt(serial, xval string) ([]byte, []byte, error) {
	if strings.Contains(xval, "-") {
		xval = strings.Replace(xval, "-", "", -1)
	}
	if len(xval) != 16 {
		return nil, nil, errors.New("Invalid X value. Without dashes, length is not 16")
	}
	if len(serial) != 0xC {
		return nil, nil, errors.New(fmt.Sprintf("Invalid console serial number. Length is not %d", 0xC))
	}

	mac := hmac.New(sha1.New, []byte(serial))
	mac.Write([]byte("XBOX360SSB"))
	desKey := mac.Sum(nil)[0:8]

	if len(desKey) != 8 {
		return nil, nil, errors.New(fmt.Sprintf("Error decrypting (invalid DES key length of %d). Key: %v", len(desKey), desKey))
	}

	xvalAsHex, _ := hex.DecodeString(xval)

	cipher, err := des.NewCipher(desKey)
	if err != nil {
		return nil, nil, err
	}
	cipher.Decrypt(xvalAsHex, xvalAsHex)
	return desKey, xvalAsHex, nil
}

// TextResult returns an array of descriptive messages for each
// flag in the dec
func TextResult(dec []byte) []string {
	var result []string

	buf := new(bytes.Buffer)
	buf.Write(dec[0:4])

	var lowBits, highBits int32
	binary.Read(buf, binary.BigEndian, &highBits)
	buf.Reset()
	buf.Write(dec[4:8])
	binary.Read(buf, binary.BigEndian, &lowBits)

	if lowBits == 0 && highBits == 0 {
		return []string{"Secdata is clean"}
	} else if lowBits == -1 && highBits == -1 {
		return []string{"Secdata is invalid"}
	} else if lowBits != 0 && highBits != 0 {
		return []string{"Secdata decryption error"}
	} else {
		for flag, message := range getFlagDescriptions() {
			if lowBits&flag != 0 {
				result = append(result, message)
			}
		}
		if lowBits&-0x7FFF != 0 {
			result = append(result, "Unknown violation(s)")
		}
	}
	return result
}
