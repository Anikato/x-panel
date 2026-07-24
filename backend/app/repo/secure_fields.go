package repo

import (
	"fmt"

	"xpanel/global"
	"xpanel/security/credentials"
)

type secureField struct {
	Scope string
	Value *string
}

func protectFields(fields ...secureField) error {
	for _, field := range fields {
		if field.Value == nil || *field.Value == "" {
			continue
		}
		if global.CREDENTIALS == nil {
			return fmt.Errorf("credential protector is unavailable for %s", field.Scope)
		}
		protected, err := global.CREDENTIALS.Protect(field.Scope, *field.Value)
		if err != nil {
			return fmt.Errorf("protect %s: %w", field.Scope, err)
		}
		*field.Value = protected
	}
	return nil
}

func revealFields(fields ...secureField) error {
	for _, field := range fields {
		if field.Value == nil || *field.Value == "" {
			continue
		}
		if global.CREDENTIALS == nil {
			return fmt.Errorf("credential protector is unavailable for %s", field.Scope)
		}
		plaintext, err := global.CREDENTIALS.Reveal(field.Scope, *field.Value)
		if err != nil {
			return fmt.Errorf("reveal %s: %w", field.Scope, err)
		}
		*field.Value = plaintext
	}
	return nil
}

func protectUpdates(table string, updates map[string]interface{}) (map[string]interface{}, error) {
	protected := make(map[string]interface{}, len(updates))
	for column, value := range updates {
		protected[column] = value
		scope, registered := credentials.ScopeFor(table, column)
		if !registered || value == nil {
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%s.%s credential update must be a string", table, column)
		}
		if text == "" {
			continue
		}
		if global.CREDENTIALS == nil {
			return nil, fmt.Errorf("credential protector is unavailable for %s", scope)
		}
		encrypted, err := global.CREDENTIALS.Protect(scope, text)
		if err != nil {
			return nil, fmt.Errorf("protect %s: %w", scope, err)
		}
		protected[column] = encrypted
	}
	return protected, nil
}
