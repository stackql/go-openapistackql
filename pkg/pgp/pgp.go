package pgp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

// gpg --export ${YOURKEYID} --export-options export-minimal,no-export-attributes | hexdump /dev/stdin -v -e '/1 "%02X"'
// gpg --export ${YOURKEYID} --export-options export-minimal,no-export-attributes | hexdump -v -e '/1 "%02X"'
var defaultPublicKeyHex string = "9833046203506016092B06010401DA470F01010740F461A2A88E14E26B8FCD1EE589AE352627E42759291B02CA229AFBAE1E9B4B63B42747656E6572616C204B726F6C6C203C67656E6572616C6B726F6C6C3040676D61696C2E636F6D3E889A0413160A00421621042B0FCDFC6041B9584A78EE1A0EDE91D385CB3D34050262035060021B03050903C26700050B090807020322020106150A09080B020416020301021E07021780000A09100EDE91D385CB3D34DE1D00FF5E0A4A37B23A5FDEB9534F47F15421B75D65E541991C5E16CEFE86BF3903292300FD117FA8CD44BB6107C9A8042A22633C5B579AA8AF29D98AFB87027CF770AA410EB8380462035060120A2B060104019755010501010740BE4EFC862EC094B8DD26CCC463057A2132D09E1561F42E98E1C1738F48A77F4A03010807887D0418160A00261621042B0FCDFC6041B9584A78EE1A0EDE91D385CB3D34050262035060021B0C050903C26700000A09100EDE91D385CB3D3444AA00F887F053438154844426865EB3243B5406ED7BC778A9BB6F48A13CCBA7842B340100E1A62629378D53F46E8AC515CE97617E1FBEF9F937C3EE5A041CD0028B190006"

func CheckSig(source io.Reader, sigr io.Reader, publicKeyHex string) ([]byte, error) {
	// First, get the content of the signed file
	if publicKeyHex == "" {
		publicKeyHex = defaultPublicKeyHex
	}
	fileContent, err := io.ReadAll(source)
	if err != nil {
		return nil, err
	}

	// Read the signature
	armor.Decode(sigr)
	pack, err := packet.Read(sigr)
	if err != nil {
		return nil, err
	}

	// Was it really a signature file ? If yes, get the Signature
	signature, ok := pack.(*packet.Signature)
	if !ok {
		return nil, fmt.Errorf("invalid signature provided")
	}

	// For convenience, we have the key in hexadecimal, convert it to binary
	publicKeyBin, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return nil, err
	}

	// Read the key
	pack, err = packet.Read(bytes.NewReader(publicKeyBin))
	if err != nil {
		return nil, err
	}

	// Was it really a public key file ? If yes, get the PublicKey
	publicKey, ok := pack.(*packet.PublicKey)
	if !ok {
		return nil, errors.New("Invalid public key.")
	}

	// Get the hash method used for the signature
	hash := signature.Hash.New()

	// Hash the content of the file (if the file is big, that's where you have to change the code to avoid getting the whole file in memory, by reading and writting in small chunks)
	_, err = hash.Write(fileContent)
	if err != nil {
		return nil, err
	}

	// Check the signature
	err = publicKey.VerifySignature(hash, signature)
	if err != nil {
		return nil, err
	}

	return fileContent, nil
}
