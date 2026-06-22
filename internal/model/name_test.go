package model

import "testing"

func TestValidateSecretName(t *testing.T) {
	t.Parallel()

	valid := []string{"myapp", "my.app", "MyApp123", "a", "app.config.v2"}
	for _, name := range valid {
		if err := ValidateSecretName(name); err != nil {
			t.Errorf("ValidateSecretName(%q) = %v, want nil", name, err)
		}
	}

	invalid := []string{"", "my-app", "my-app/config", "foo bar", "foo_bar", "foo@bar", "foo/bar"}
	for _, name := range invalid {
		if err := ValidateSecretName(name); err == nil {
			t.Errorf("ValidateSecretName(%q) = nil, want error", name)
		}
	}
}
