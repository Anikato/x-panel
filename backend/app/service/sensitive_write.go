package service

import "fmt"

type settingValue struct {
	Key   string
	Value string
}

func persistSettingValues(
	write func(string, string) error,
	values ...settingValue,
) error {
	for _, value := range values {
		if err := write(value.Key, value.Value); err != nil {
			return fmt.Errorf("persist setting %s: %w", value.Key, err)
		}
	}
	return nil
}
