package api

import "testing"

func TestValidPIN(t *testing.T) {
	cases := []struct{ pin string; want bool }{
		{"123456",false},{"654321",false},{"000000",false},{"111111",false},
		{"121212",true},{"258036",true},{"482915",true},{"12345",false},{"12a456",false},
	}
	for _,tc:=range cases { if got:=validPIN(tc.pin); got!=tc.want { t.Errorf("validPIN(%q)=%v, want %v",tc.pin,got,tc.want) } }
}

func TestOnboardingCountryAllowlist(t *testing.T) {
	for _,code:=range []string{"DE","FR","ES","NL"} { if _,ok:=findOnboardingCountry(code); !ok { t.Errorf("%s should be supported",code) } }
	for _,code:=range []string{"RU","BY","US","GB"} { if _,ok:=findOnboardingCountry(code); ok { t.Errorf("%s should not be in the initial EU phone-region allowlist",code) } }
}

func TestE164Validation(t *testing.T) {
	for _,phone:=range []string{"+4915112345678","+33123456789","+34612345678"} { if !e164Pattern.MatchString(phone) { t.Errorf("%s should match E.164",phone) } }
	for _,phone:=range []string{"004915112345678","4915112345678","+123","++49123456789"} { if e164Pattern.MatchString(phone) { t.Errorf("%s should not match E.164",phone) } }
}
