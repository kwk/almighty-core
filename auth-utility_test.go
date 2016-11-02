package main_test

import (
	"fmt"
	"testing"
	"time"

	. "github.com/almighty/almighty-core"
	jwt "github.com/dgrijalva/jwt-go"
)

const (

	// the various HTTP endpoints
	EndpointWorkItems              = "/api/workitems"
	EndpointWorkItemTypes          = "/api/workitemtypes"
	EndpointWorkItemLinkCategories = "/api/workitemlinkcategories"
)

// GetExpiredAuthHeader returns a JWT bearer token with an expiration date that lies in the past
func GetExpiredAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	token.Claims = jwt.MapClaims{"exp": float64(time.Now().Unix() - 100)}
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Bearer %s", tokenStr)
}

// GetMalformedAuthHeader returns a JWT bearer token with the wrong prefix of "Malformed Bearer" instead of just "Bearer"
func GetMalformedAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Malformed Bearer %s", tokenStr)
}

// GetExpiredAuthHeader returns a valid JWT bearer token
func GetValidAuthHeader(t *testing.T, key interface{}) string {
	token := jwt.New(jwt.SigningMethodRS256)
	tokenStr, err := token.SignedString(key)
	if err != nil {
		t.Fatal("Could not sign the token ", err)
	}
	return fmt.Sprintf("Bearer %s", tokenStr)
}

// The RSADifferentPrivateKeyTest key will be used to sign the token but verification should
// fail as this is not the key used by server security layer
// ssh-keygen -f test-alm
var RSADifferentPrivateKeyTest = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsIT76Mr3p8VvtSrzCVcXEcyvalUp50mm4yvfqvZ1fZfbqAzJ
c4GNJEkpBGoXF+WgjLNkPnwS+k1YuqvPeGG4vFPtErF7nxNCHpzU+cXScOOl3WrM
S5fj928sBSJiDIDBwc98mQbIKaCrpLSsFe/kapV5mHmmWGAx6dqnObbqtIte4M7w
arE/c8xW1Fww2YZ4e4Xwknm+Rs2kQmg0SJPpgRih05y3snEQjXz1kR9bGTEBUPmX
HBTySgA93gmimQUlSAT0+hz9hcYrwCgjbnXUHlcBP2VbB4omJ7L1zJc/XMPwINR/
PtkGRhL/DXXA4v/0MLkYDXXmZGku/X1+du2ypQIDAQABAoIBAFi0m3si9E2FNFvQ
l42sDFXPjJ9c6M/n/UvP8niRnf1dYO8Ube/zvJ/tfAVR4wUJSiMqy0dzRn4ufFZi
nMIcKZ/KdSqdskgAf4uuuIBEXzqHzAR29O9QBymC3pY97xPlaHki8bRc6h2xNlBw
0sG7agf90btD9soWnT6tuLeSKmRLh5aHUQv3aGwzPyNfKHQ8J/KwKdPudP+tVBsi
eNd7DZDgSEw6pRaSCKS3ChrsQQ2XPjGo+OI6HjZ/LAFhFXMq2cRGELGF766a6phK
aCTB619AXiRHdKE98zEY3GMDtXSgeA0yzxcbvr224rEkHGDfkZ0BJwOCqCiaw4tZ
F/lFDMkCgYEA36Uqyj0cML5rMwC/W4b6ihuK/DujBBFYPQ8eVYt5yUSyLNJn5RLt
33eBUvgGB/FyAio5aCp49mcPtfFv5GKXpzTSYo/bWV1iy+oVwgPF7UA5gvtRw90w
NScLNsZ/7fOEpPJvlsKq/PQoMIoAjkegoj95cqM1yzC3aZpaAjx6188CgYEAyg58
5e5WK3zXICMpE8q+1AB+kJ/3UhQ71kpK4Xml0PtTJ7Bzqn0hiU4ThfpKj1n9PtpU
9Op3PqcfVjf11SA1tI5LRHQvgUSNppvf2hPgW8QrqRs5YFgNg0DkVXs3OxWIA4QA
Ko6oZu2ZpvK3TdYXRmcRUXXNyCDoSmJvH339N0sCgYB0g1kCmcm4/0tb+/S1m2Gl
V+oVtIAeG2csEFdOW+ar27Uzsr5b0nvI4zql3f9OXhR2WkckJJR2UoUV1d3kTxUR
EGzW2nl9WjChaafCNzMDgmUz/vi/INn/pwKpm8qETkz5njBSi8KHHDBf8VWOynQ+
cvEzryHUZOH5C2f/KEEbcwKBgQCGzVGgaPjOPJSdWTfPf4T+lXHa9Q4gkWU2WwxI
D0uD+BiLMxqH1MGqBA/cY5aYutXMuAbT+xUhFIhAkkcNMFcEJaarfcQvvteuHvIi
YP5e2qqyQHpv/27McV+kc/buEThT+B3QRqqtOLk4+1c1s66Fhr+0FB789I9lCPTQ
EtL7rwKBgQC5x7lGs+908uqf7yFXHzw7rPGFUe6cuxZ3jVOzovVoXRma+C7nroNx
/A4rWPqfpmiKcmrd7K4DQFlYhoq+MALEDmQm+/8G6j2inF53fRGgJVzaZhSvnO9X
CMnDipW5SU9AQE+xC8Zc+02rcyuZ7ha1WXKgIKwAa92jmJSCJjzdxA==
-----END RSA PRIVATE KEY-----`
