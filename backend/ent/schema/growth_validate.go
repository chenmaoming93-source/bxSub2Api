package schema

import "fmt"

func growthEnum(fieldName string, allowed ...string) func(string) error {
	return func(value string) error {
		for _, candidate := range allowed {
			if value == candidate {
				return nil
			}
		}
		return fmt.Errorf("%s must be one of %v", fieldName, allowed)
	}
}

func growthNonNegative(fieldName string) func(int64) error {
	return func(value int64) error {
		if value < 0 {
			return fmt.Errorf("%s must be non-negative", fieldName)
		}
		return nil
	}
}

func growthPositive(fieldName string) func(int64) error {
	return func(value int64) error {
		if value <= 0 {
			return fmt.Errorf("%s must be positive", fieldName)
		}
		return nil
	}
}

func growthNonNegativeInt(fieldName string) func(int) error {
	return func(value int) error {
		if value < 0 {
			return fmt.Errorf("%s must be non-negative", fieldName)
		}
		return nil
	}
}

func growthPositiveInt(fieldName string) func(int) error {
	return func(value int) error {
		if value <= 0 {
			return fmt.Errorf("%s must be positive", fieldName)
		}
		return nil
	}
}
