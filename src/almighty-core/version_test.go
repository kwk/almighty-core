package main

import (
	"testing"

	"github.com/almighty/almighty-core/app/test"
)

func TestAuthorizeLoginOK(t *testing.T) {
	controller := LoginController{}
	resp := test.AuthorizeLoginOK(t, &controller)

	if resp.Token == "" {
		t.Error("Token not generated")
	}
}

func TestShowVersionOK(t *testing.T) {
	controller := VersionController{}
	resp := test.ShowVersionOK(t, &controller)

	if resp.Commit != "0" {
		t.Error("Commit not found")
	}
}

var ValidJWTToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NjA4MzUyNzUsInNjb3BlcyI6WyJzeXN0ZW0iXX0.OHsz9bIN9nKemd8Rdm9lYapXOknh5nwvCN8ZD_YIVfCZ54MkoKiIjj_VsGclRMCykDtXD4Omg2mWuiaEDPoP4nHRjlWfup3Us29k78cpImBz6FwfK08J39pKr0Y7s-Qdpq_XGwdTEWx7Hk33nrgyZVdMfE4nRjCulkIWbhOxNDdjKqUSo3zknRQRWzZhVl8a1cMNG6EetFHe-pCEr3WpreeRZcoL948smll_16WYB8r3t2-jtW7CmrJwSx7ZMopD-AvOaAGsiExgNRUd5YcSX0zEl5mjwnSb-rqemQt4_BHs0zgufyDw5MtH0ZG8phNIbyWt3G1VaO3CqDt_Ixxh7Q"

var InValidJWTToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE0NjA4MzUyNzUsInNjb3BlcyI6WyJzeXN0ZW0iXX0.OHsz9bIN9nKemd8Rdm9lYapXOknh5nwvCN8ZD_YIVfCZ54MkoKiIjj_VsGclRMCykDtXD4Omg2mWuiaEDPoP4nHRjlWfup3Us29k78cpImBz6FwfK08J39pKr0Y7s-Qdpq_XGwdTEWx7Hk33nrgyZVdMfE4nRjCulkIWbhOxNDdjKqUSo3zknRQRWzZhVl8a1cMNG6EetFHe-pCEr3WpreeRZcoL948smll_16WYB8r3t2-jtW7CmrJwSx7ZMopD-AvOaAGsiExgNRUd5YcSX0zEl5mjwnSb-rqemQt4_BHs0zgufyDw5MtH0ZG8phNIbyWt3G1VaO3CqDt_Ixxh7"
